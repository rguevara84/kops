# etcd-manager Certificate Expiration

etcd-manager configures certificates for TLS communication between kube-apiserver and etcd, as well as between etcd members.
These certificates are signed by the cluster CA and are valid for a duration of 1 year.

Affected versions of etcd-manager currently do NOT automatically rotate these certificates before expiration.
If these certificates are not rotated prior to their expiration, Kubernetes apiserver will become inaccessible and your control-plane will experience downtime.

## How do I know if I'm affected?

Clusters are affected by this issue if they're using a version of etcd-manager < `3.0.20200428`.
The etcd-manager version is set automatically based on the kOps version.
These kOps versions are affected:

* kOps 1.10.0-alpha.1 through 1.15.2
* kOps 1.16.0-alpha.1 through 1.16.1
* kOps 1.17.0-alpha.1 through 1.17.0-beta.1
* kOps 1.18.0-alpha.1 through 1.18.0-alpha.2

The issue can be confirmed by checking for the existence of etcd-manager pods and observing their image tags:

```bash
kubectl get pods -n kube-system -l k8s-app=etcd-manager-main \
  -o jsonpath='{range .items[*].spec.containers[*]}{.image}{"\n"}{end}'
```

* If this outputs `kopeio/etcd-manager` images with tags older than `3.0.20200428`, the cluster is affected.
* If this outputs an image other than `kopeio/etcd-manager`, the cluster may be affected.
* If this does does not output anything or outputs `kopeio/etcd-manager` images with tags >= `3.0.20200428`, the cluster is not affected.

The issue can be confirmed also by checking the certificate expiry using `openssl` on each master node.

```bash
find /mnt/ -type f -name me.crt -print -exec openssl x509 -enddate -noout -in {} \;
```

## Solution

Upgrade etcd-manager. etcd-manager version >= `3.0.20200428` manages certificate lifecycle and will automatically request new certificates before expiration.

We have two suggested workflows to upgrade etcd-manager in your cluster. While both workflows require a rolling-update of the master nodes, neither require control-plane downtime (if the clusters have highly available masters).

1. Upgrade to kOps 1.15.3, 1.16.2, 1.17.0-beta.2, or 1.18.0-alpha.3.
   This is the recommended approach.
   Follow the normal steps when upgrading kOps and confirm the etcd-manager image will be updated based on the output of `kops update cluster`.
   ```
   kops update cluster --yes
   kops rolling-update cluster --instance-group-roles=Master --cloudonly
   ```
2. Another solution is to override the etcd-manager image in the ClusterSpec.
   The image will be set in two places, one for each etcdCluster (main and events).
   ```
   kops edit cluster $CLUSTER_NAME
   # Set `spec.etcdClusters[*].manager.image` to `kopeio/etcd-manager:3.0.20200428`
   kops update cluster # confirm the image is being updated
   kops update cluster --yes
   kops rolling-update cluster --instance-group-roles=Master --force --cloudonly
   ```
