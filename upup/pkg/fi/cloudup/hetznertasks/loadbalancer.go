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

package hetznertasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/hetzner"
)

// +kops:fitask
type LoadBalancer struct {
	Name      *string
	Lifecycle fi.Lifecycle
	Network   *Network

	ID       *int
	Location string
	Type     string
	Services []*LoadBalancerService
	Target   string

	Labels map[string]string
}

var _ fi.CompareWithID = &LoadBalancer{}

func (v *LoadBalancer) CompareWithID() *string {
	return fi.String(strconv.Itoa(fi.IntValue(v.ID)))
}

var _ fi.HasAddress = &LoadBalancer{}

func (e *LoadBalancer) IsForAPIServer() bool {
	return true
}

func (v *LoadBalancer) FindAddresses(c *fi.Context) ([]string, error) {
	cloud := c.Cloud.(hetzner.HetznerCloud)
	client := cloud.LoadBalancerClient()

	// TODO(hakman): Find using label selector
	loadbalancers, err := client.All(context.TODO())
	if err != nil {
		return nil, err
	}

	for _, loadbalancer := range loadbalancers {
		if loadbalancer.Name == fi.StringValue(v.Name) {
			var addresses []string
			addresses = append(addresses, loadbalancer.PublicNet.IPv4.IP.String())
			for _, privateNetwork := range loadbalancer.PrivateNet {
				addresses = append(addresses, privateNetwork.IP.String())
			}
			return addresses, nil
		}
	}

	return nil, nil
}

func (v *LoadBalancer) Find(c *fi.Context) (*LoadBalancer, error) {
	cloud := c.Cloud.(hetzner.HetznerCloud)
	client := cloud.LoadBalancerClient()

	// TODO(hakman): Find using label selector
	loadbalancers, err := client.All(context.TODO())
	if err != nil {
		return nil, err
	}

	for _, loadbalancer := range loadbalancers {
		if loadbalancer.Name == fi.StringValue(v.Name) {
			matches := &LoadBalancer{
				Lifecycle: v.Lifecycle,
				Name:      fi.String(loadbalancer.Name),
				ID:        fi.Int(loadbalancer.ID),
				Labels:    loadbalancer.Labels,
			}

			if loadbalancer.Location != nil {
				matches.Location = loadbalancer.Location.Name
			}
			if loadbalancer.LoadBalancerType != nil {
				matches.Type = loadbalancer.LoadBalancerType.Name
			}

			for _, service := range loadbalancer.Services {
				loadbalancerService := LoadBalancerService{
					Protocol:        string(service.Protocol),
					ListenerPort:    fi.Int(service.ListenPort),
					DestinationPort: fi.Int(service.DestinationPort),
				}
				matches.Services = append(matches.Services, &loadbalancerService)
			}

			for _, target := range loadbalancer.Targets {
				if target.Type == hcloud.LoadBalancerTargetTypeLabelSelector && target.LabelSelector != nil {
					matches.Target = target.LabelSelector.Selector
				}
			}

			// TODO: The API only returns the network ID, a new API call is required to get the network name
			matches.Network = v.Network

			v.ID = matches.ID
			return matches, nil
		}
	}

	return nil, nil
}

func (v *LoadBalancer) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(v, c)
}

func (_ *LoadBalancer) CheckChanges(a, e, changes *LoadBalancer) error {
	if a != nil {
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
		if changes.ID != nil {
			return fi.CannotChangeField("ID")
		}
		if changes.Location != "" {
			return fi.CannotChangeField("Location")
		}
		if changes.Type != "" {
			return fi.CannotChangeField("Type")
		}
		if len(changes.Services) > 0 && len(a.Services) > 0 {
			return fi.CannotChangeField("Subnets")
		}
		if changes.Target != "" && a.Target != "" {
			return fi.CannotChangeField("Target")
		}
	} else {
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
		if e.Location == "" {
			return fi.RequiredField("Location")
		}
		if e.Type == "" {
			return fi.RequiredField("Type")
		}
		if len(e.Services) == 0 {
			return fi.RequiredField("Services")
		}
		if e.Target == "" {
			return fi.RequiredField("Target")
		}
	}
	return nil
}

func (_ *LoadBalancer) RenderHetzner(t *hetzner.HetznerAPITarget, a, e, changes *LoadBalancer) error {
	client := t.Cloud.LoadBalancerClient()

	var loadbalancer *hcloud.LoadBalancer
	if a == nil {
		if e.Network == nil {
			return fmt.Errorf("failed to find network for loadbalancer %q", fi.StringValue(e.Name))
		}

		opts := hcloud.LoadBalancerCreateOpts{
			Name: fi.StringValue(e.Name),
			LoadBalancerType: &hcloud.LoadBalancerType{
				Name: e.Type,
			},
			Algorithm: &hcloud.LoadBalancerAlgorithm{
				Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
			},
			Location: &hcloud.Location{
				Name: e.Location,
			},
			Labels: e.Labels,
			Services: []hcloud.LoadBalancerCreateOptsService{
				{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      fi.Int(443),
					DestinationPort: fi.Int(443),
				},
			},
			Network: &hcloud.Network{
				ID: fi.IntValue(e.Network.ID),
			},
		}
		result, _, err := client.Create(context.TODO(), opts)
		if err != nil {
			return err
		}
		loadbalancer = result.LoadBalancer

	} else {
		var err error
		loadbalancer, _, err = client.Get(context.TODO(), strconv.Itoa(fi.IntValue(a.ID)))
		if err != nil {
			return err
		}

		// Update the labels
		if changes.Name != nil || len(changes.Labels) != 0 {
			_, _, err := client.Update(context.TODO(), loadbalancer, hcloud.LoadBalancerUpdateOpts{
				Name:   fi.StringValue(e.Name),
				Labels: e.Labels,
			})
			if err != nil {
				return err
			}
		}

		// Update the services
		if len(changes.Services) > 0 {
			for _, service := range e.Services {
				_, _, err := client.AddService(context.TODO(), loadbalancer, hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocol(service.Protocol),
					ListenPort:      service.ListenerPort,
					DestinationPort: service.DestinationPort,
				})
				if err != nil {
					if err != nil {
						return err
					}
				}
			}
		}

	}

	// Add the target separately, otherwise UsePrivateIP cannot be set
	// https://github.com/hetznercloud/hcloud-go/pull/198
	if a == nil || a.Target == "" {
		_, _, err := client.AddLabelSelectorTarget(context.TODO(), loadbalancer, hcloud.LoadBalancerAddLabelSelectorTargetOpts{
			Selector:     e.Target,
			UsePrivateIP: fi.Bool(true),
		})
		if err != nil {
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadBalancerService represents a LoadBalancer's service.
type LoadBalancerService struct {
	Protocol        string
	ListenerPort    *int
	DestinationPort *int
}

var _ fi.HasDependencies = &LoadBalancerService{}

func (e *LoadBalancerService) GetDependencies(tasks map[string]fi.Task) []fi.Task {
	return nil
}
