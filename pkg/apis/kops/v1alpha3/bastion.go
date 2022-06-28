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

type BastionSpec struct {
	// PublicName is the domain name for the bastion load balancer.
	PublicName string `json:"publicName,omitempty"`
	// IdleTimeoutSeconds is the bastion's load balancer idle timeout.
	IdleTimeoutSeconds *int64 `json:"idleTimeoutSeconds,omitempty"`
	// LoadBalancer contains settings for the load balancer fronting bastion instances.
	LoadBalancer *BastionLoadBalancerSpec `json:"loadBalancer,omitempty"`
}

type BastionLoadBalancerSpec struct {
	AdditionalSecurityGroups []string `json:"additionalSecurityGroups,omitempty"`
	// Type of load balancer to create, it can be Public or Internal.
	Type LoadBalancerType `json:"type,omitempty"`
}
