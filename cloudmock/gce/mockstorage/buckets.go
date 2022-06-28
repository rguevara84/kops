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

package mockstorage

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"google.golang.org/api/storage/v1"
	"k8s.io/kops/cloudmock/gce/gcphttp"
)

type buckets struct {
	mutex sync.Mutex

	policies map[string]*storage.Policy
}

func (s *buckets) Init() {
	s.policies = make(map[string]*storage.Policy)
}

func (s *buckets) getIAMPolicy(bucket string, request *http.Request) (*http.Response, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policies[bucket]
	if policy == nil {
		policy = &storage.Policy{}
	}

	return gcphttp.OKResponse(policy)
}

func (s *buckets) setIAMPolicy(bucket string, request *http.Request) (*http.Response, error) {
	b, err := io.ReadAll(request.Body)
	if err != nil {
		return gcphttp.ErrorBadRequest("")
	}

	req := &storage.Policy{}
	if err := json.Unmarshal(b, &req); err != nil {
		return gcphttp.ErrorBadRequest("")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// TODO: etag

	policy := req
	s.policies[bucket] = policy

	return gcphttp.OKResponse(policy)
}
