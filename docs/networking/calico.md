# Calico

[Calico](https://docs.projectcalico.org/latest/introduction/) is an open source networking and
network security solution for containers, virtual machines, and native host-based workloads.

Calico combines flexible networking capabilities with run-anywhere security enforcement to provide
a solution with native Linux kernel performance and true cloud-native scalability. Calico provides
developers and cluster operators with a consistent experience and set of capabilities whether
running in public cloud or on-prem, on a single node or across a multi-thousand node cluster.

## Installing

To use the Calico, specify the following in the cluster spec.

```yaml
  networking:
    calico: {}
```

The following command sets up a cluster using Calico.

```sh
export ZONES=mylistofzones
kops create cluster \
  --zones $ZONES \
  --networking calico \
  --yes \
  --name myclustername.mydns.io
```

## Configuring

### Select an Encapsulation Mode

In order to send network traffic to and from Kubernetes pods, Calico can use either of two networking encapsulation modes: [IP-in-IP](https://tools.ietf.org/html/rfc2003)  or [VXLAN](https://tools.ietf.org/html/rfc7348). Though IP-in-IP encapsulation uses fewer bytes of overhead per packet than VXLAN encapsulation, [VXLAN can be a better
choice when used in concert with Calico's eBPF dataplane|https://docs.projectcalico.org/maintenance/troubleshoot/troubleshoot-ebpf#poor-performance]. In particular, eBPF programs can redirect packets between Layer 2 devices, but not between devices at Layer 2 and Layer 3, as is required to use IP-in-IP tunneling.

kOps chooses the IP-in-IP encapsulation mode by default, it still being the Calico project's default choice, which is equivalent to writing the following in the cluster spec:
```yaml
  networking:
    calico:
      encapsulationMode: ipip
```
To use the VXLAN encapsulation mode instead, add the following to the cluster spec:
```yaml
  networking:
    calico:
      encapsulationMode: vxlan
```

As of Calico version 3.17, in order to use IP-in-IP encapsulation, Calico must use its BIRD networking backend, in which it runs the BIRD BGP daemon in each "calico-node" container to distribute routes to each machine. With the BIRD backend Calico can use either IP-in-IP or VXLAN encapsulation between machines. For now, IP-in-IP encapsulation requires maintaining the routes with BGP, whereas VXLAN encapsulation does not. Conversely, with the VXLAN backend, Calico does not run the BIRD daemon and does not use BGP to maintain routes. This rules out use of IP-in-IP encapsulation, and allows only VXLAN encapsulation. Calico may remove this need for BGP with IP-in-IP encapsulation in the future.

### Enable Cross-Subnet mode in Calico

Calico supports a new option for both of its IP-in-IP and VXLAN encapsulation modes where traffic is only encapsulated
when it’s destined to subnets with intermediate infrastructure lacking Calico route awareness, for example, across
heterogeneous public clouds or on AWS where traffic is crossing availability zones.

With this mode, encapsulation is only [performed selectively](https://docs.projectcalico.org/v3.10/networking/vxlan-ipip#configure-ip-in-ip-encapsulation-for-only-cross-subnet-traffic).
This provides better performance in AWS multi-AZ deployments, or those spanning multiple VPC subnets within a single AZ, and in general when deploying on networks where pools of nodes with L2 connectivity are connected via a router.

Note that by default with Calico—when using its BIRD networking backend—routes between nodes within a subnet are
distributed using a full node-to-node BGP mesh.
Each node automatically sets up a BGP peering with every other node within the same L2 network.
This full node-to-node mesh per L2 network has its scaling challenges for larger scale deployments.
BGP route reflectors can be used as a replacement to a full mesh, and is useful for scaling up a cluster. [BGP route reflectors are recommended once the number of nodes goes above ~50-100.](https://docs.projectcalico.org/networking/bgp#topologies-for-public-cloud)
The setup of BGP route reflectors is currently out of the scope of kOps.

Read more here: [BGP route reflectors](https://docs.projectcalico.org/reference/architecture/overview#bgp-route-reflector-bird)

To enable this mode in a cluster, add the following to the cluster spec:

```yaml
  networking:
    calico:
      crossSubnet: true
```
In the case of AWS, EC2 instances' ENIs have source/destination checks enabled by default.
When you enable cross-subnet mode in kOps 1.19+, it is equivalent to either:
```yaml
  networking:
    calico:
      awsSrcDstCheck: Disable
      IPIPMode: CrossSubnet
```
or
```yaml
  networking:
    calico:
      awsSrcDstCheck: Disable
      encapsulationMode: vxlan
```
depending on which encapsulation mode you have selected.

**Cross-subnet mode is the default mode in kOps 1.22+** for both IP-in-IP and VXLAN encapsulation.
It can be disabled or adjusted by setting the `ipipMode`, `vxlanMode` and `awsSrcDstCheck` options.

In AWS an IAM policy will be added to all nodes to allow Calico to execute `ec2:DescribeInstances` and `ec2:ModifyNetworkInterfaceAttribute`, as required when [awsSrcDstCheck](https://docs.projectcalico.org/reference/resources/felixconfig#spec) is set.
For older versions of kOps, an addon controller ([k8s-ec2-srcdst](https://github.com/ottoyiu/k8s-ec2-srcdst))
will be deployed as a Pod (which will be scheduled on one of the masters) to facilitate the disabling of said source/destination address checks.
Only the control plane nodes have an IAM policy to allow k8s-ec2-srcdst to execute `ec2:ModifyInstanceAttribute`.

### Configuring Calico MTU

The Calico MTU is configurable by editing the cluster and setting `mtu` field in the Calico configuration. If left to its default empty value, Calico will inspect the network devices and [choose a suitable MTU value automatically](https://docs.projectcalico.org/networking/mtu#mtu-and-calico-defaults). If you decide to override this automatic tuning, specify a positive value for the `mtu` field. In AWS, VPCs support jumbo frames of size 9,001, so [the recommended choice for Calico's MTU](https://docs.projectcalico.org/networking/mtu#determine-mtu-size) is either 8,981 for IP-in-IP encapsulation, 8,951 for VXLAN encapsulation, or 8,941 for WireGuard, in each case deducting the appropriate overhead for the encapsulation format.

```yaml
spec:
  networking:
    calico:
      mtu: 8981
```

### Configuring Calico to use Typha

As of kOps 1.12 Calico uses the kube-apiserver as its datastore. The default setup does not make use of [Typha](https://github.com/projectcalico/typha)—a component intended to lower the impact of Calico on the Kubernetes API Server which is recommended in [clusters over 50 nodes](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/calico#installing-with-the-kubernetes-api-datastoremore-than-50-nodes) and is strongly recommended in clusters of 100+ nodes.
It is possible to configure Calico to use Typha by editing a cluster and adding the `typhaReplicas` field with a positive value to the Calico spec:

```yaml
  networking:
    calico:
      typhaReplicas: 3
```

### Configuring the eBPF dataplane
{{ kops_feature_table(kops_added_default='1.19', k8s_min='1.16') }}

Calico supports using an [eBPF dataplane](https://docs.projectcalico.org/about/about-ebpf) as an alternative to the standard Linux dataplane (which is based on iptables). While the standard dataplane focuses on compatibility by relying on kube-proxy and your own iptables rules, the eBPF dataplane focuses on performance, latency, and improving user experience with features that aren’t possible in the standard dataplane. As part of that, the eBPF dataplane replaces kube-proxy with an eBPF implementation. The main “user experience” feature is to preserve the source IP address of traffic from outside the cluster when traffic hits a node port; this makes the server-side logs and network policy much more useful on that path.

For more details on enabling the eBPF dataplane please refer the [Calico Docs](https://docs.projectcalico.org/maintenance/ebpf/enabling-bpf).

Enable the eBPF dataplane in kOps—while also disabling use of kube-proxy—as follows:

```yaml
  kubeProxy:
    enabled: false
  networking:
    calico:
      bpfEnabled: true
```

You can further tune Calico's eBPF dataplane with additional options, such as enabling [DSR mode](https://docs.projectcalico.org/maintenance/enabling-bpf#try-out-dsr-mode) to eliminate network hops in node port traffic (feasible only when your cluster conforms to [certain restrictions](https://docs.projectcalico.org/maintenance/troubleshoot/troubleshoot-ebpf#troubleshoot-access-to-services)) or [increasing the log verbosity for Calico's eBPF programs](https://docs.projectcalico.org/maintenance/troubleshoot/troubleshoot-ebpf#ebpf-program-debug-logs):

```yaml
  kubeProxy:
    enabled: false
  networking:
    calico:
      bpfEnabled: true
      bpfExternalServiceMode: DSR
      bpfLogLevel: Debug
```

**Note:** Transitioning to or from Calico's eBPF dataplane in an existing cluster is disruptive. kOps cannot orchestrate this transition automatically today.

### Configuring WireGuard
{{ kops_feature_table(kops_added_default='1.19', k8s_min='1.16') }}

Calico supports WireGuard to encrypt pod-to-pod traffic. If you enable this options, WireGuard encryption is automatically enabled for all nodes. At the moment, kOps installs WireGuard automatically only when the host OS is *Ubuntu*. For other OSes, WireGuard has to be part of the base image or installed via a hook.

For more details of Calico WireGuard please refer the [Calico Docs](https://docs.projectcalico.org/security/encrypt-cluster-pod-traffic).

```yaml
  networking:
    calico:
      wireguardEnabled: true
```

## Getting help

For help with Calico or to report any issues:
* [Calico Github](https://github.com/projectcalico/calico)
* [Calico Users Slack](https://calicousers.slack.com)

For more general information on options available with Calico see the official [Calico docs](https://docs.projectcalico.org/latest/introduction/):
* See [Calico Network Policy](https://docs.projectcalico.org/latest/security/calico-network-policy)
  for details on the additional features not available with Kubernetes Network Policy.
* See [Determining best Calico networking option](https://docs.projectcalico.org/latest/networking/determine-best-networking)
  for help with the network options available with Calico.



## Troubleshooting

### New nodes are taking minutes for syncing IP routes and new pods on them can't reach kubedns

This is caused by nodes in the Calico etcd nodestore no longer existing. Due to the ephemeral nature of AWS EC2 instances, new nodes are brought up with different hostnames, and nodes that are taken offline remain in the Calico nodestore. This is unlike most datacentre deployments where the hostnames are mostly static in a cluster. Read this issue](https://github.com/kubernetes/kops/issues/3224) for more detail.

This has been solved in kOps 1.9.0, when creating a new cluster no action is needed, but if the cluster was created with a prior kOps version the following actions should be taken:

  * Use kOps to update the cluster ```kops update cluster <name> --yes``` and wait for calico-kube-controllers deployment and calico-node daemonset pods to be updated
  * Decommission all invalid nodes, [see here](https://docs.projectcalico.org/v2.6/usage/decommissioning-a-node)
  * All nodes that are deleted from the cluster after this actions should be cleaned from calico's etcd storage and the delay programming routes should be solved.
