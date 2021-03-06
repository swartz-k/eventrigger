/*
Copyright 2021.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActorTemplate is the template that describes trigger specification.
type ActorTemplate struct {
	// Name is a unique name of the action to take.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Conditions is the conditions to execute the trigger.
	// For example: "(dep01 || dep02) && dep04"
	// +optional
	Conditions string `json:"conditions,omitempty" protobuf:"bytes,2,opt,name=conditions"`
	// StandardK8STrigger refers to the trigger designed to create or update a generic Kubernetes resource.
	// +optional
	K8s *StandardK8SActor `json:"k8s,omitempty" protobuf:"bytes,3,opt,name=k8s"`

	HTTP *HTTPActor `json:"http,omitempty" protobuf:"bytes,5,opt,name=http"`
	// AWSLambda refers to the trigger designed to invoke AWS Lambda function with with on-the-fly constructable payload.
	// +optional
}

// Actor is an action taken, output produced, an events created, a message sent
type Actor struct {
	// Template describes the trigger specification.
	Template *ActorTemplate `json:"template,omitempty" protobuf:"bytes,1,opt,name=template"`
}

// SensorSpec defines the desired state of Sensor
type SensorSpec struct {
	// Foo is an example field of Sensor. Edit sensor_types.go to remove/update
	Trigger Trigger `json:"trigger"  protobuf:"bytes,1,name=trigger" yaml:"trigger"`
	// Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor.
	Actor Actor `json:"actor" protobuf:"bytes,2,rep,name=actor" yaml:"actor"`

	Target Target `json:"target" protobuf:"bytes,3,rep,name=target" yaml:"target"`
}

// SensorStatus defines the observed state of Sensor
type SensorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status `json:",inline" protobuf:"bytes,1,opt,name=status"`
}

// Sensor is the definition of a sensor resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=sn
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type Sensor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec SensorSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// +optional
	Status SensorStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// SensorList is the list of Sensor resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SensorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sensor `json:"items"`
}
