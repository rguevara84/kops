/*
Copyright 2019 The Kubernetes Authors.

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

package cloudup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/kops/pkg/testutils"

	kopsapi "k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/assets"
	"k8s.io/kops/pkg/client/simple/vfsclientset"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awsup"
	"k8s.io/kops/util/pkg/vfs"
)

func buildMinimalCluster() (*awsup.MockAWSCloud, *kopsapi.Cluster) {
	cloud := awsup.InstallMockAWSCloud(MockAWSRegion, "abcd")

	c := testutils.BuildMinimalCluster("testcluster.test.com")

	return cloud, c
}

func TestPopulateCluster_Default_NoError(t *testing.T) {
	cloud, c := buildMinimalCluster()

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	_, err = mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}
}

func TestPopulateCluster_Subnets(t *testing.T) {
	tests := []struct {
		NonMasqueradeCIDR             string
		ExpectedClusterCIDR           string
		ExpectedServiceClusterIPRange string
	}{
		{
			NonMasqueradeCIDR:             "100.64.0.0/10",
			ExpectedClusterCIDR:           "100.96.0.0/11",
			ExpectedServiceClusterIPRange: "100.64.0.0/13",
		},
		{
			NonMasqueradeCIDR:             "10.0.0.0/9",
			ExpectedClusterCIDR:           "10.64.0.0/10",
			ExpectedServiceClusterIPRange: "10.0.0.0/12",
		},
		{
			NonMasqueradeCIDR:             "10.0.0.0/8",
			ExpectedClusterCIDR:           "10.128.0.0/9",
			ExpectedServiceClusterIPRange: "10.0.0.0/12",
		},
		{
			NonMasqueradeCIDR:             "::/0",
			ExpectedServiceClusterIPRange: "fd00:5e4f:ce::/108",
		},
	}
	for _, tc := range tests {
		t.Run(tc.NonMasqueradeCIDR, func(t *testing.T) {
			cloud, c := buildMinimalCluster()
			c.Spec.NonMasqueradeCIDR = tc.NonMasqueradeCIDR
			c.Spec.Networking.Kubenet = nil
			c.Spec.Networking.CNI = &kopsapi.CNINetworkingSpec{}
			c.Spec.ExternalCloudControllerManager = &kopsapi.CloudControllerManagerConfig{}
			c.Spec.CloudConfig = &kopsapi.CloudConfiguration{
				AWSEBSCSIDriver: &kopsapi.AWSEBSCSIDriver{
					Enabled: fi.Bool(true),
				},
			}

			err := PerformAssignments(c, cloud)
			require.NoError(t, err, "PerformAssignments")

			full, err := mockedPopulateClusterSpec(c, cloud)
			require.NoError(t, err, "PopulateClusterSpec")

			assert.Equal(t, tc.ExpectedClusterCIDR, full.Spec.KubeControllerManager.ClusterCIDR, "ClusterCIDR")
			assert.Equal(t, tc.ExpectedServiceClusterIPRange, full.Spec.ServiceClusterIPRange, "ServiceClusterIPRange")
		})
	}
}

func mockedPopulateClusterSpec(c *kopsapi.Cluster, cloud fi.Cloud) (*kopsapi.Cluster, error) {
	vfs.Context.ResetMemfsContext(true)

	assetBuilder := assets.NewAssetBuilder(c, false)
	basePath, err := vfs.Context.BuildVfsPath("memfs://tests")
	if err != nil {
		return nil, fmt.Errorf("error building vfspath: %v", err)
	}
	clientset := vfsclientset.NewVFSClientset(basePath)
	return PopulateClusterSpec(clientset, c, cloud, assetBuilder)
}

func TestPopulateCluster_Docker_Spec(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.Docker = &kopsapi.DockerConfig{
		MTU:                fi.Int32(5678),
		InsecureRegistry:   fi.String("myregistry.com:1234"),
		InsecureRegistries: []string{"myregistry.com:1234", "myregistry2.com:1234"},
		RegistryMirrors:    []string{"https://registry.example.com"},
		LogOpt:             []string{"env=FOO"},
	}

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}

	if fi.Int32Value(full.Spec.Docker.MTU) != 5678 {
		t.Fatalf("Unexpected Docker MTU: %v", full.Spec.Docker.MTU)
	}

	if fi.StringValue(full.Spec.Docker.InsecureRegistry) != "myregistry.com:1234" {
		t.Fatalf("Unexpected Docker InsecureRegistry: %v", full.Spec.Docker.InsecureRegistry)
	}

	if strings.Join(full.Spec.Docker.InsecureRegistries, "!") != "myregistry.com:1234!myregistry2.com:1234" {
		t.Fatalf("Unexpected Docker InsecureRegistries: %v", full.Spec.Docker.InsecureRegistries)
	}

	if strings.Join(full.Spec.Docker.RegistryMirrors, "!") != "https://registry.example.com" {
		t.Fatalf("Unexpected Docker RegistryMirrors: %v", full.Spec.Docker.RegistryMirrors)
	}

	if strings.Join(full.Spec.Docker.LogOpt, "!") != "env=FOO" {
		t.Fatalf("Unexpected Docker LogOpt: %v", full.Spec.Docker.LogOpt)
	}
}

func TestPopulateCluster_StorageDefault(t *testing.T) {
	cloud, c := buildMinimalCluster()

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}

	if fi.StringValue(full.Spec.KubeAPIServer.StorageBackend) != "etcd3" {
		t.Fatalf("Unexpected StorageBackend: %v", *full.Spec.KubeAPIServer.StorageBackend)
	}
}

func build(c *kopsapi.Cluster) (*kopsapi.Cluster, error) {
	cloud, err := BuildCloud(c)
	if err != nil {
		return nil, fmt.Errorf("error from BuildCloud: %v", err)
	}

	err = PerformAssignments(c, cloud)
	if err != nil {
		return nil, fmt.Errorf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		return nil, fmt.Errorf("Unexpected error from PopulateCluster: %v", err)
	}
	return full, nil
}

func TestPopulateCluster_Custom_CIDR(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.NetworkCIDR = "172.20.2.0/24"
	c.Spec.Subnets = []kopsapi.ClusterSubnetSpec{
		{Name: "subnet-us-mock-1a", Zone: "us-mock-1a", CIDR: "172.20.2.0/27", Type: kopsapi.SubnetTypePublic},
		{Name: "subnet-us-mock-1b", Zone: "us-mock-1b", CIDR: "172.20.2.32/27", Type: kopsapi.SubnetTypePublic},
		{Name: "subnet-us-mock-1c", Zone: "us-mock-1c", CIDR: "172.20.2.64/27", Type: kopsapi.SubnetTypePublic},
	}

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}
	if full.Spec.NetworkCIDR != "172.20.2.0/24" {
		t.Fatalf("Unexpected NetworkCIDR: %v", full.Spec.NetworkCIDR)
	}
}

func TestPopulateCluster_IsolateMasters(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.IsolateMasters = fi.Bool(true)

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}
	if fi.BoolValue(full.Spec.MasterKubelet.EnableDebuggingHandlers) != false {
		t.Fatalf("Unexpected EnableDebuggingHandlers: %v", fi.BoolValue(full.Spec.MasterKubelet.EnableDebuggingHandlers))
	}
	if fi.BoolValue(full.Spec.MasterKubelet.ReconcileCIDR) != false {
		t.Fatalf("Unexpected ReconcileCIDR: %v", fi.BoolValue(full.Spec.MasterKubelet.ReconcileCIDR))
	}
}

func TestPopulateCluster_IsolateMastersFalse(t *testing.T) {
	cloud, c := buildMinimalCluster()
	// default: c.Spec.IsolateMasters = fi.Bool(false)

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}
	if fi.BoolValue(full.Spec.MasterKubelet.EnableDebuggingHandlers) != true {
		t.Fatalf("Unexpected EnableDebuggingHandlers: %v", fi.BoolValue(full.Spec.MasterKubelet.EnableDebuggingHandlers))
	}
}

func TestPopulateCluster_Name_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.ObjectMeta.Name = ""

	expectErrorFromPopulateCluster(t, c, cloud, "Name")
}

func TestPopulateCluster_Zone_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.Subnets = nil

	expectErrorFromPopulateCluster(t, c, cloud, "subnet")
}

func TestPopulateCluster_NetworkCIDR_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.NetworkCIDR = ""

	expectErrorFromPopulateCluster(t, c, cloud, "networkCIDR")
}

func TestPopulateCluster_NonMasqueradeCIDR_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.NonMasqueradeCIDR = ""

	expectErrorFromPopulateCluster(t, c, cloud, "nonMasqueradeCIDR")
}

func TestPopulateCluster_CloudProvider_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.CloudProvider = kopsapi.CloudProviderSpec{}

	expectErrorFromPopulateCluster(t, c, cloud, "cloudProvider")
}

func TestPopulateCluster_TopologyInvalidNil_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.Topology.Masters = ""
	c.Spec.Topology.Nodes = ""
	expectErrorFromPopulateCluster(t, c, cloud, "topology")
}

func TestPopulateCluster_TopologyInvalidValue_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.Topology.Masters = "123"
	c.Spec.Topology.Nodes = "abc"
	expectErrorFromPopulateCluster(t, c, cloud, "topology")
}

//func TestPopulateCluster_TopologyInvalidMatchingValues_Required(t *testing.T) {
//	// We can't have a bastion with public masters / nodes
//	c := buildMinimalCluster()
//	c.Spec.Topology.Masters = api.TopologyPublic
//	c.Spec.Topology.Nodes = api.TopologyPrivate
//	expectErrorFromPopulateCluster(t, c, "Topology")
//}

func TestPopulateCluster_BastionInvalidMatchingValues_Required(t *testing.T) {
	// We can't have a bastion with public masters / nodes
	cloud, c := buildMinimalCluster()
	c.Spec.Topology.Masters = kopsapi.TopologyPublic
	c.Spec.Topology.Nodes = kopsapi.TopologyPublic
	c.Spec.Topology.Bastion = &kopsapi.BastionSpec{}
	expectErrorFromPopulateCluster(t, c, cloud, "bastion")
}

func TestPopulateCluster_BastionIdleTimeoutInvalidNegative_Required(t *testing.T) {
	cloud, c := buildMinimalCluster()

	c.Spec.Topology.Masters = kopsapi.TopologyPrivate
	c.Spec.Topology.Nodes = kopsapi.TopologyPrivate
	c.Spec.Topology.Bastion = &kopsapi.BastionSpec{}
	c.Spec.Topology.Bastion.IdleTimeoutSeconds = fi.Int64(-1)
	expectErrorFromPopulateCluster(t, c, cloud, "bastion")
}

func expectErrorFromPopulateCluster(t *testing.T, c *kopsapi.Cluster, cloud fi.Cloud, message string) {
	_, err := mockedPopulateClusterSpec(c, cloud)
	if err == nil {
		t.Fatalf("Expected error from PopulateCluster")
	}
	actualMessage := fmt.Sprintf("%v", err)
	if !strings.Contains(actualMessage, message) {
		t.Fatalf("Expected error %q, got %q", message, actualMessage)
	}
}

func TestPopulateCluster_APIServerCount(t *testing.T) {
	_, c := buildMinimalCluster()

	full, err := build(c)
	if err != nil {
		t.Fatalf("error during build: %v", err)
	}

	if fi.Int32Value(full.Spec.KubeAPIServer.APIServerCount) != 3 {
		t.Fatalf("Unexpected APIServerCount: %v", fi.Int32Value(full.Spec.KubeAPIServer.APIServerCount))
	}
}

func TestPopulateCluster_AnonymousAuth(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.KubernetesVersion = "1.20.0"

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}

	if full.Spec.KubeAPIServer.AnonymousAuth == nil {
		t.Fatalf("AnonymousAuth not specified")
	}

	if fi.BoolValue(full.Spec.KubeAPIServer.AnonymousAuth) != false {
		t.Fatalf("Unexpected AnonymousAuth: %v", fi.BoolValue(full.Spec.KubeAPIServer.AnonymousAuth))
	}
}

func TestPopulateCluster_DockerVersion(t *testing.T) {
	grid := []struct {
		KubernetesVersion string
		DockerVersion     string
	}{
		{
			KubernetesVersion: "1.17.0",
			DockerVersion:     "19.03.15",
		},
		{
			KubernetesVersion: "1.21.0",
			DockerVersion:     "20.10.17",
		},
	}

	for _, test := range grid {
		_, c := buildMinimalCluster()
		c.Spec.KubernetesVersion = test.KubernetesVersion
		c.Spec.ContainerRuntime = "docker"

		full, err := build(c)
		if err != nil {
			t.Fatalf("error during build: %v", err)
		}

		if fi.StringValue(full.Spec.Docker.Version) != test.DockerVersion {
			t.Fatalf("Unexpected DockerVersion: %v", fi.StringValue(full.Spec.Docker.Version))
		}
	}
}

func TestPopulateCluster_KubeController_High_Enough_Version(t *testing.T) {
	cloud, c := buildMinimalCluster()
	c.Spec.KubernetesVersion = "v1.9.0"

	err := PerformAssignments(c, cloud)
	if err != nil {
		t.Fatalf("error from PerformAssignments: %v", err)
	}

	full, err := mockedPopulateClusterSpec(c, cloud)
	if err != nil {
		t.Fatalf("Unexpected error from PopulateCluster: %v", err)
	}

	if full.Spec.KubeControllerManager.AttachDetachReconcileSyncPeriod == nil {
		t.Fatalf("AttachDetachReconcileSyncPeriod not set correctly")
	}
}
