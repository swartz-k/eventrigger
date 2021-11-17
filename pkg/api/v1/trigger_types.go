package v1

import (
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// KubernetesResourceOperation refers to the type of operation performed on the K8s resource
type KubernetesResourceOperation string

// possible values for KubernetesResourceOperation
const (
	Create KubernetesResourceOperation = "create" // create the resource
	Update KubernetesResourceOperation = "update" // updates the resource
	Patch  KubernetesResourceOperation = "patch"  // patch resource
	Delete KubernetesResourceOperation = "delete" // deletes the resource
)

// StandardK8STrigger is the standard Kubernetes resource trigger
type StandardK8STrigger struct {
	// Operation refers to the type of operation performed on the k8s resource.
	// Default value is Create.
	// +optional
	Operation KubernetesResourceOperation `json:"operation,omitempty" protobuf:"bytes,1,opt,name=operation,casttype=KubernetesResourceOperation"`
	// PatchStrategy controls the K8s object patching strategy when the trigger operation is specified as patch.
	// possible values:
	// "application/json-patch+json"
	// "application/merge-patch+json"
	// "application/strategic-merge-patch+json"
	// "application/apply-patch+yaml".
	// Defaults to "application/merge-patch+json"
	// +optional
	PatchStrategy k8stypes.PatchType `json:"patchStrategy,omitempty" protobuf:"bytes,2,opt,name=patchStrategy,casttype=k8s.io/apimachinery/pkg/types.PatchType"`
	// LiveObject specifies whether the resource should be directly fetched from K8s instead
	// of being marshaled from the resource artifact. If set to true, the resource artifact
	// must contain the information required to uniquely identify the resource in the cluster,
	// that is, you must specify "apiVersion", "kind" as well as "name" and "namespace" meta
	// data.
	// Only valid for operation type `update`
	// +optional
	LiveObject bool `json:"liveObject,omitempty" protobuf:"varint,3,opt,name=liveObject"`
}

type HTTPTrigger struct {
	// URL refers to the URL to send HTTP request to.
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`

	// Method refers to the type of the HTTP request.
	// Refer https://golang.org/src/net/http/method.go for more info.
	// Default value is POST.
	// +optional
	Method string `json:"method,omitempty" protobuf:"bytes,2,opt,name=method"`
	// Timeout refers to the HTTP request timeout in seconds.
	// Default value is 60 seconds.
	// +optional
	Timeout int64 `json:"timeout,omitempty" protobuf:"varint,3,opt,name=timeout"`
	// Headers for the HTTP request.
	// +optional
	Headers map[string]string `json:"headers,omitempty" protobuf:"bytes,4,rep,name=headers"`
}
