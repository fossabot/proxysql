/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	ResourceCodeEtcdVersion     = "etcversion"
	ResourceKindEtcdVersion     = "EtcdVersion"
	ResourceSingularEtcdVersion = "etcdversion"
	ResourcePluralEtcdVersion   = "etcdversions"
)

// EtcdVersion defines a Etcd database version.

// +genclient
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=etcdversions,singular=etcdversion,scope=Cluster,shortName=etcversion,categories={datastore,kubedb,appscode}
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="DB_IMAGE",type="string",JSONPath=".spec.db.image"
// +kubebuilder:printcolumn:name="Deprecated",type="boolean",JSONPath=".spec.deprecated"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type EtcdVersion struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              EtcdVersionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// EtcdVersionSpec is the spec for postgres version
type EtcdVersionSpec struct {
	// Version
	Version string `json:"version" protobuf:"bytes,1,opt,name=version"`
	// Database Image
	DB EtcdVersionDatabase `json:"db" protobuf:"bytes,2,opt,name=db"`
	// Exporter Image
	Exporter EtcdVersionExporter `json:"exporter" protobuf:"bytes,3,opt,name=exporter"`
	// Tools Image
	Tools EtcdVersionTools `json:"tools" protobuf:"bytes,4,opt,name=tools"`
	// Deprecated versions usable but regarded as obsolete and best avoided, typically due to having been superseded.
	// +optional
	Deprecated bool `json:"deprecated,omitempty" protobuf:"varint,5,opt,name=deprecated"`
}

// EtcdVersionDatabase is the Etcd Database image
type EtcdVersionDatabase struct {
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
}

// EtcdVersionExporter is the image for the Etcd exporter
type EtcdVersionExporter struct {
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
}

// EtcdVersionTools is the image for the Etcd exporter
type EtcdVersionTools struct {
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EtcdVersionList is a list of EtcdVersions
type EtcdVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Items is a list of EtcdVersion CRD objects
	Items []EtcdVersion `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
