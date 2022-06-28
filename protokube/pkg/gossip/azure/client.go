/*
Copyright 2020 The Kubernetes Authors.

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

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type instanceComputeMetadata struct {
	Name              string `json:"name"`
	ResourceGroupName string `json:"resourceGroupName"`
	SubscriptionID    string `json:"subscriptionId"`
	// Tags is a list of tags separated by ';'. Each tag
	// is of the form "key:value".
	Tags string `json:"tags"`
}

func (m *instanceComputeMetadata) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	l := strings.Split(m.Tags, ";")
	for _, t := range l {
		tl := strings.Split(t, ":")
		if len(tl) != 2 {
			return nil, fmt.Errorf("unexpected tag format: %s", tl)
		}
		tags[tl[0]] = tl[1]
	}
	return tags, nil
}

type ipAddress struct {
	PrivateIPAddress string `json:"privateIpAddress"`
	PublicIPAddress  string `json:"publicIpAddress"`
}

type ipv4Interface struct {
	IPAddresses []*ipAddress `json:"ipAddress"`
}

type networkInterface struct {
	IPv4 *ipv4Interface `json:"ipv4"`
}

type instanceNetworkMetadata struct {
	Interfaces []*networkInterface `json:"interface"`
}

type instanceMetadata struct {
	Compute *instanceComputeMetadata `json:"compute"`
	Network *instanceNetworkMetadata `json:"network"`
}

// Client is an Azure client.
type Client struct {
	metadata         *instanceMetadata
	vmssesClient     *compute.VirtualMachineScaleSetsClient
	interfacesClient *network.InterfacesClient
}

// NewClient returns a new Client.
func NewClient() (*Client, error) {
	m, err := queryInstanceMetadata()
	if err != nil {
		return nil, fmt.Errorf("error querying instance metadata: %s", err)
	}
	if m.Compute.SubscriptionID == "" {
		return nil, fmt.Errorf("empty subscription name")
	}
	if m.Compute.ResourceGroupName == "" {
		return nil, fmt.Errorf("empty resource group name")
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("error creating an authorizer: %s", err)
	}

	vmssesClient := compute.NewVirtualMachineScaleSetsClient(m.Compute.SubscriptionID)
	vmssesClient.Authorizer = authorizer

	interfacesClient := network.NewInterfacesClient(m.Compute.SubscriptionID)
	interfacesClient.Authorizer = authorizer

	return &Client{
		metadata:         m,
		vmssesClient:     &vmssesClient,
		interfacesClient: &interfacesClient,
	}, nil
}

func (c *Client) resourceGroupName() string {
	return c.metadata.Compute.ResourceGroupName
}

// GetName returns the name of the VM.
func (c *Client) GetName() string {
	return c.metadata.Compute.Name
}

// GetTags returns the tags of the VM queried from Instance Metadata Service.
func (c *Client) GetTags() (map[string]string, error) {
	return c.metadata.Compute.GetTags()
}

// GetInternalIP returns the internal IP of the VM queried from Instance Metadata Service.
// This function returns nil if no internal IP is found.
func (c *Client) GetInternalIP() net.IP {
	for _, iface := range c.metadata.Network.Interfaces {
		if iface.IPv4 == nil {
			continue
		}
		for _, ipAddr := range iface.IPv4.IPAddresses {
			if a := ipAddr.PrivateIPAddress; a != "" {
				return net.ParseIP(a)
			}
		}
	}
	return nil
}

// ListVMScaleSets returns VM ScaleSets in the resource group.
func (c *Client) ListVMScaleSets(ctx context.Context) ([]compute.VirtualMachineScaleSet, error) {
	var l []compute.VirtualMachineScaleSet
	for iter, err := c.vmssesClient.ListComplete(ctx, c.resourceGroupName()); iter.NotDone(); err = iter.Next() {
		if err != nil {
			return nil, err
		}
		l = append(l, iter.Value())
	}
	return l, nil
}

// ListVMSSNetworkInterfaces returns the interfaces that the specified VM ScaleSet has.
func (c *Client) ListVMSSNetworkInterfaces(ctx context.Context, vmScaleSetName string) ([]network.Interface, error) {
	var l []network.Interface
	for iter, err := c.interfacesClient.ListVirtualMachineScaleSetNetworkInterfacesComplete(ctx, c.resourceGroupName(), vmScaleSetName); iter.NotDone(); err = iter.Next() {
		if err != nil {
			return nil, err
		}
		l = append(l, iter.Value())
	}
	return l, nil
}

// queryInstanceMetadata queries Azure Instance Metadata documented in
// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service.
func queryInstanceMetadata() (*instanceMetadata, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating a new request: %s", err)
	}
	req.Header.Add("Metadata", "True")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2020-06-01")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to the metadata server: %s", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading a response from the metadata server: %s", err)
	}
	metadata, err := unmarshalInstanceMetadata(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling metadata: %s", err)
	}
	return metadata, nil
}

func unmarshalInstanceMetadata(data []byte) (*instanceMetadata, error) {
	m := &instanceMetadata{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
