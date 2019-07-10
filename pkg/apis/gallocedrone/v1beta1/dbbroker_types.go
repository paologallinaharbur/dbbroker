/*
Copyright 2019 Paolo.Gallina.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DbBrokerSpec defines the desired state of DbBroker
type DbBrokerSpec struct {
	DeploymentName      string `json:"deploymentName,omitempty"`
	DeploymentNameSpace string `json:"deploymentNamespace,omitempty"`
	ProjectID           string `json:"projectID,omitempty"`
}

// DbBrokerStatus defines the observed state of DbBroker
type DbBrokerStatus struct {
	Initialised bool   `json:"initialised,omitempty"`
	Username    string `json:"username,omitempty"`
	EndPoint    string `json:"endPoint,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbBroker is the Schema for the dbbrokers API
// +k8s:openapi-gen=true
type DbBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbBrokerSpec   `json:"spec,omitempty"`
	Status DbBrokerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbBrokerList contains a list of DbBroker
type DbBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbBroker{}, &DbBrokerList{})
}
