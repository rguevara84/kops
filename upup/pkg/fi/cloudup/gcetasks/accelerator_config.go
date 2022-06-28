/*
Copyright 2022 The Kubernetes Authors.

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

package gcetasks

import (
	"fmt"

	"k8s.io/kops/upup/pkg/fi"
)

// AcceleratorConfig defines an accelerator config
type AcceleratorConfig struct {
	AcceleratorCount int64  `json:"acceleratorCount,omitempty"`
	AcceleratorType  string `json:"acceleratorType,omitempty"`
}

var (
	_ fi.HasDependencies = &AcceleratorConfig{}
)

func (a *AcceleratorConfig) GetDependencies(tasks map[string]fi.Task) []fi.Task {
	return nil
}

func (_ *AcceleratorConfig) ShouldCreate(a, e, changes *AcceleratorConfig) (bool, error) {
	if e.AcceleratorCount < 0 {
		return false, fmt.Errorf("acceleratorCount must be positive or 0")
	}
	if e.AcceleratorType == "" {
		return false, fmt.Errorf("acceleratorType must not be empty")
	}
	return true, nil
}
