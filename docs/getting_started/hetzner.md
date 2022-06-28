# Getting Started with kOps on Hetzner Cloud

**WARNING**: Hetzner Cloud support on kOps is currently in **alpha**, meaning it is subject to change, so please use with caution.
The original issue ticket is [#8983](https://github.com/kubernetes/kops/issues/8983).

## Requirements
* kOps version >= 1.24
* kubectl version >= 1.23
* Hetzner Cloud [account](https://accounts.hetzner.com/login)
* Hetzner Cloud [token](https://docs.hetzner.cloud/#authentication)
* SSH public and private keys
* S3 compatible object storage (like [MinIO](https://docs.min.io/minio/baremetal/security/minio-identity-management/user-management.html))

## Environment Variables

It is important to set the following environment variables:
```bash
export KOPS_FEATURE_FLAGS=Hetzner
export HCLOUD_TOKEN=<token>
export S3_ENDPOINT=<endpoint>
export S3_ACCESS_KEY_ID=<acces-key>
export S3_SECRET_ACCESS_KEY=<secret-key>
export KOPS_STATE_STORE=s3://<bucket-name>
```

## Creating a Single Master Cluster

In the following examples, `example.k8s.local` is a [gossip-based DNS ](../gossip.md) cluster name.

```bash
# create a ubuntu 20.04 + calico cluster in fsn1
kops create cluster --name=my-cluster.example.k8s.local \
  --ssh-public-key=~/.ssh/id_rsa.pub --cloud=hetzner --zones=fsn1 \
  --image=ubuntu-20.04 --networking=calico --network-cidr=10.10.0.0/16 
kops update cluster my-cluster.example.k8s.local --yes

# create a ubuntu 20.04 + calico cluster in fsn1 with CPU optimized servers
kops create cluster --name=my-cluster.example.k8s.local \
  --ssh-public-key=~/.ssh/id_rsa.pub --cloud=hetzner --zones=fsn1 \
  --image=ubuntu-20.04 --networking=calico --network-cidr=10.10.0.0/16 \
  --node-size cpx31
kops update cluster my-cluster.example.k8s.local --yes

# delete a cluster
kops delete cluster --name=my-cluster.example.k8s.local --yes

# export kubecfg
# See https://kops.sigs.k8s.io/cli/kops_export_kubeconfig/#examples. 

# update a cluster
# See https://kops.sigs.k8s.io/operations/updates_and_upgrades/#manual-update.
```

## Features Still in Development

kOps for Hetzner Cloud currently does not support the following features:

* Cluster validation
* Rolling updates
* Autoscaling using [Cluster Autoscaler](https://github.com/hetznercloud/autoscaler)
* Volumes using the [CSI Driver](https://github.com/hetznercloud/csi-driver)
* [Terraform](https://github.com/hetznercloud/terraform-provider-hcloud) support
* Multiple SSH keys 

# Next steps

Now that you have a working kOps cluster, read through the recommendations for [production setups guide](production.md) to learn more about how to configure kOps for production workloads.
