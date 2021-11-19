package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// S3Bucket contains information to describe an S3 Bucket
type S3Bucket struct {
	Key  string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

// S3Filter represents filters to apply to bucket notifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// S3Artifact contains information about an S3 connection and bucket
type S3Artifact struct {
	Endpoint  string                    `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket    *S3Bucket                 `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`
	Region    string                    `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Insecure  bool                      `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
	AccessKey *corev1.SecretKeySelector `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`
	SecretKey *corev1.SecretKeySelector `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`

	Events   []string          `json:"events,omitempty" protobuf:"bytes,7,rep,name=events"`
	Filter   *S3Filter         `json:"filter,omitempty" protobuf:"bytes,8,opt,name=filter"`
	Metadata map[string]string `json:"metadata,omitempty" protobuf:"bytes,9,opt,name=metadata"`
}

// FileArtifact contains information about an artifact in a filesystem
type FileArtifact struct {
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}

// URLArtifact contains information about an artifact at an http endpoint.
type URLArtifact struct {
	// Path is the complete URL
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`
	// VerifyCert decides whether the connection is secure or not
	VerifyCert bool `json:"verifyCert,omitempty" protobuf:"varint,2,opt,name=verifyCert"`
}
