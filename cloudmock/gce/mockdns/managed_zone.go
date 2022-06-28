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

package mockdns

import (
	dns "google.golang.org/api/dns/v1"
	"k8s.io/kops/upup/pkg/fi/cloudup/gce"
)

type managedZoneClient struct {
	// managedZones are managedZones keyed by project and managedZone name.
	managedZones map[string]map[string]*dns.ManagedZone
}

var _ gce.ManagedZoneClient = &managedZoneClient{}

func newManagedZoneClient() *managedZoneClient {
	return &managedZoneClient{
		managedZones: map[string]map[string]*dns.ManagedZone{},
	}
}

func (c *managedZoneClient) List(project string) ([]*dns.ManagedZone, error) {
	mzs, ok := c.managedZones[project]
	if !ok {
		return nil, nil
	}
	var l []*dns.ManagedZone
	for _, mz := range mzs {
		l = append(l, mz)
	}
	return l, nil
}
