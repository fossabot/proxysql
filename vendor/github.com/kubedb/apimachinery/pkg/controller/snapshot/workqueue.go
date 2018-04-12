package snapshot

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Controller) initWatcher() {

	// create the workqueue
	c.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot")

	// Watch with label selector
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.ExtClient.Snapshots(metav1.NamespaceAll).List(c.listOption)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.ExtClient.Snapshots(metav1.NamespaceAll).Watch(c.listOption)
		},
	}

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the Snapshot key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Snapshot than the version which was responsible for triggering the update.
	c.indexer, c.informer = cache.NewIndexerInformer(lw, &api.Snapshot{}, c.syncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			snapshot, ok := obj.(*api.Snapshot)
			if !ok {
				log.Errorln("Invalid Snapshot object")
				return
			}
			if snapshot.Status.StartTime == nil {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					c.queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			snapshot, ok := obj.(*api.Snapshot)
			if !ok {
				log.Errorln("Invalid Snapshot object")
				return
			}
			if snapshot.DeletionTimestamp != nil {
				key, err := cache.MetaNamespaceKeyFunc(snapshot)
				if err == nil {
					c.queue.Add(key)
				}
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (c *Controller) runWatcher(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Infoln("Starting Snapshot Controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Infoln("Stopping Snapshot Controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two Snapshots with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.runSnapshot(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		log.Debugf("Finished Processing key: %v\n", key)
		return true
	}
	log.Errorf("Failed to process Snapshot %v. Reason: %s\n", key, err)

	// This Controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < c.maxNumRequests {
		log.Infof("Error syncing crd %v: %v\n", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return true
	}

	c.queue.Forget(key)
	log.Debugf("Finished Processing key: %v\n", key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Infof("Dropping Snapshot %q out of the queue: %v\n", key, err)
	return true
}

func (c *Controller) runSnapshot(key string) error {
	log.Debugf("started processing, key: %v\n", key)
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		log.Errorf("Fetching object with key %s from store failed with %v\n", key, err)
		return err
	}

	if !exists {
		log.Debugf("Snapshot %s does not exist anymore\n", key)
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Snapshot was recreated with the same name
		snapshot := obj.(*api.Snapshot).DeepCopy()
		if snapshot.DeletionTimestamp != nil {
			if core_util.HasFinalizer(snapshot.ObjectMeta, api.GenericKey) {
				util.AssignTypeKind(snapshot)
				if err := c.delete(snapshot); err != nil {
					log.Errorln(err)
					return err
				}
				snapshot, _, err = util.PatchSnapshot(c.ExtClient, snapshot, func(in *api.Snapshot) *api.Snapshot {
					in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, api.GenericKey)
					return in
				})
				return err
			}
		} else {
			snapshot, _, err = util.PatchSnapshot(c.ExtClient, snapshot, func(in *api.Snapshot) *api.Snapshot {
				in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, api.GenericKey)
				return in
			})
			util.AssignTypeKind(snapshot)
			if err := c.create(snapshot); err != nil {
				log.Errorln(err)
				return err
			}
		}
	}
	return nil
}