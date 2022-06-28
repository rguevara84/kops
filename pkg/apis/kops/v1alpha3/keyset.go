/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeysetType describes the type of keys in a KeySet
type KeysetType string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Keyset is a set of system keypairs, or other secret material.
// It is a set to support credential rotation etc.
type Keyset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KeysetSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeysetList is a list of Keysets
type KeysetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Keyset `json:"items"`
}

// KeysetItem is an item (keypair or other secret material) in a Keyset
type KeysetItem struct {
	// Id is the unique identifier for this key in the keyset
	Id string `json:"id,omitempty"`

	// DistrustTimestamp is RFC 3339 date and time at which this keypair was distrusted.
	// If not set, keypair is trusted or is not a keypair.
	DistrustTimestamp *metav1.Time `json:"distrustTimestamp,omitempty"`

	// PublicMaterial holds non-secret material (e.g. a certificate)
	PublicMaterial []byte `json:"publicMaterial,omitempty"`

	// PrivateMaterial holds secret material (e.g. a private key, or symmetric token)
	PrivateMaterial []byte `json:"privateMaterial,omitempty"`
}

// KeysetSpec is the spec for a Keyset
type KeysetSpec struct {
	// Type is the type of the Keyset (PKI keypair, or secret token)
	Type KeysetType `json:"type,omitempty"`

	// PrimaryID is the id of the key used to make new signatures.
	PrimaryID string `json:"primaryID,omitempty"`

	// Keys is the set of keys that make up the keyset
	Keys []KeysetItem `json:"keys,omitempty"`
}
