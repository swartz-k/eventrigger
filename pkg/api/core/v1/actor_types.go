package v1

import (
	"eventrigger.com/operator/pkg/api/core/common"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// KubernetesResourceOperation refers to the type of operation performed on the K8s resource
type KubernetesResourceOperation string

// possible values for KubernetesResourceOperation
const (
	Create KubernetesResourceOperation = "create" // create the resource every event
	Update KubernetesResourceOperation = "update" // updates the resource
	Patch  KubernetesResourceOperation = "patch"  // patch resource
	Delete KubernetesResourceOperation = "delete" // deletes the resource
	Scale  KubernetesResourceOperation = "scale"  // scale resource to zero
)

// StandardK8SActor is the standard Kubernetes resource trigger
type StandardK8SActor struct {
	// Source of the K8s resource file(s)
	Source *ArtifactLocation `json:"source,omitempty" protobuf:"bytes,1,opt,name=source"`
	// Operation refers to the type of operation performed on the k8s resource.
	// Default value is Create.
	// +optional
	Operation KubernetesResourceOperation `json:"operation,omitempty" protobuf:"bytes,2,opt,name=operation,casttype=KubernetesResourceOperation"`
	// PatchStrategy controls the K8s object patching strategy when the trigger operation is specified as patch.
	// possible values:
	// "application/json-patch+json"
	// "application/merge-patch+json"
	// "application/strategic-merge-patch+json"
	// "application/apply-patch+yaml".
	// Defaults to "application/merge-patch+json"
	// +optional
	PatchStrategy k8stypes.PatchType `json:"patchStrategy,omitempty" protobuf:"bytes,3,opt,name=patchStrategy,casttype=k8s.io/apimachinery/pkg/types.PatchType"`
	// ScaleToZeroTime whether to scale to zero if  now - last event receive >=  scaleToZeroTime second
	ScaleToZeroTime int32 `json:"scaleToZeroTime,omitempty" protobuf:"bytes,4,opt,name=scaleToZeroTime"`
	// ScaleMaxReplica whether to scale to zero if  now - last event receive >=  scaleToZeroTime second
	ScaleMaxReplica int32 `json:"scaleMaxReplica,omitempty" protobuf:"bytes,5,opt,name=scaleMaxReplica"`
	// ScaleMinReplica whether to scale to zero if  now - last event receive >=  scaleToZeroTime second
	ScaleMinReplica int32 `json:"scaleMinReplica,omitempty" protobuf:"bytes,6,opt,name=scaleMinReplica"`
	// LiveObject specifies whether the resource should be directly fetched from K8s instead
	// of being marshaled from the resource artifact. If set to true, the resource artifact
	// must contain the information required to uniquely identify the resource in the cluster,
	// that is, you must specify "apiVersion", "kind" as well as "name" and "namespace" meta
	// data.
	// Only valid for operation type `update`
	// +optional
	LiveObject bool `json:"liveObject,omitempty" protobuf:"varint,7,opt,name=liveObject"`
}

type HTTPActor struct {
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

// ArtifactLocation describes the source location for an external artifact
type ArtifactLocation struct {
	// S3 compliant artifact
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,1,opt,name=s3"`
	// Inline artifact is embedded in sensor spec as a string
	Inline *string `json:"inline,omitempty" protobuf:"bytes,2,opt,name=inline"`
	// File artifact is artifact stored in a file
	File *FileArtifact `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`
	// URL to fetch the artifact from
	URL *URLArtifact `json:"url,omitempty" protobuf:"bytes,4,opt,name=url"`
	// Configmap that stores the artifact
	Configmap *corev1.ConfigMapKeySelector `json:"configmap,omitempty" protobuf:"bytes,5,opt,name=configmap"`
	// Resource is generic template for K8s resource
	Resource *common.Resource `json:"resource,omitempty" protobuf:"bytes,6,opt,name=resource"`
}
