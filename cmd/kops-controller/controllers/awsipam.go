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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kops/upup/pkg/fi"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewAWSIPAMReconciler is the constructor for a IPAMReconciler
func NewAWSIPAMReconciler(mgr manager.Manager) (*AWSIPAMReconciler, error) {
	klog.Info("Starting aws ipam controller")
	r := &AWSIPAMReconciler{
		client: mgr.GetClient(),
		log:    ctrl.Log.WithName("controllers").WithName("IPAM"),
	}

	coreClient, err := corev1client.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("error building corev1 client: %v", err)
	}
	r.coreV1Client = coreClient

	config := aws.NewConfig()
	config = config.WithCredentialsChainVerboseErrors(true)

	s, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("error starting new AWS session: %v", err)
	}
	s.Handlers.Send.PushFront(func(r *request.Request) {
		// Log requests
		klog.V(4).Infof("AWS API Request: %s/%s", r.ClientInfo.ServiceName, r.Operation.Name)
	})

	metadata := ec2metadata.New(s, config)

	region, err := metadata.Region()
	if err != nil {
		return nil, fmt.Errorf("error querying ec2 metadata service (for region): %v", err)
	}

	r.ec2Client = ec2.New(s, config.WithRegion(region))

	return r, nil
}

// AWSIPAMReconciler observes Node objects, and labels them with the correct labels for the instancegroup
// This used to be done by the kubelet, but is moving to a central controller for greater security in 1.16
type AWSIPAMReconciler struct {
	// client is the controller-runtime client
	client client.Client

	// log is a logr
	log logr.Logger

	// coreV1Client is a client-go client for patching nodes
	coreV1Client *corev1client.CoreV1Client

	ec2Client *ec2.EC2
}

// +kubebuilder:rbac:groups=,resources=nodes,verbs=get;list;watch;patch
// Reconcile is the main reconciler function that observes node changes.
func (r *AWSIPAMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.log.WithValues("ipam-controller", req.NamespacedName)

	node := &corev1.Node{}
	if err := r.client.Get(ctx, req.NamespacedName, node); err != nil {
		klog.Warningf("unable to fetch node %s: %v", node.Name, err)
		if apierrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if len(node.Spec.PodCIDRs) == 0 {
		// CCM Node Controller has not done its thing yet
		if node.Spec.ProviderID == "" {
			klog.Infof("Node %q has empty provider ID", node.Name)
			return ctrl.Result{}, nil
		}

		// aws:///eu-central-1a/i-07577a7bcf3e576f2
		providerURL, err := url.Parse(node.Spec.ProviderID)
		if err != nil {
			return ctrl.Result{}, err
		}
		instanceID := strings.Split(providerURL.Path, "/")[2]
		eni, err := r.ec2Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
			Filters: []*ec2.Filter{
				{
					Name: fi.String("attachment.instance-id"),
					Values: []*string{
						&instanceID,
					},
				},
			},
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		if len(eni.NetworkInterfaces) != 1 {
			return ctrl.Result{}, fmt.Errorf("unexpected number of network interfaces for instance %q: %v", instanceID, len(eni.NetworkInterfaces))
		}

		if len(eni.NetworkInterfaces[0].Ipv6Prefixes) != 1 {
			return ctrl.Result{}, fmt.Errorf("unexpected amount of ipv6 prefixes on interface %q: %v", *eni.NetworkInterfaces[0].NetworkInterfaceId, len(eni.NetworkInterfaces[0].Ipv6Prefixes))
		}

		patchNodePodCIDRs(r.coreV1Client, ctx, node, *eni.NetworkInterfaces[0].Ipv6Prefixes[0].Ipv6Prefix)

	}

	return ctrl.Result{}, nil
}

func (r *AWSIPAMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}

type nodePatchSpec struct {
	PodCIDR  string   `json:"podCIDR,omitempty"`
	PodCIDRs []string `json:"podCIDRs,omitempty"`
}

// patchNodeLabels patches the node labels to set the specified labels
func patchNodePodCIDRs(client *corev1client.CoreV1Client, ctx context.Context, node *corev1.Node, cidr string) error {
	klog.Infof("assigning cidr %q to node %q", cidr, node.ObjectMeta.Name)
	nodePatchSpec := &nodePatchSpec{
		PodCIDR:  cidr,
		PodCIDRs: []string{cidr},
	}
	nodePatch := &nodePatch{
		Spec: nodePatchSpec,
	}
	nodePatchJson, err := json.Marshal(nodePatch)
	if err != nil {
		return fmt.Errorf("error building node patch: %v", err)
	}

	klog.V(2).Infof("sending patch for node %q: %q", node.Name, string(nodePatchJson))

	_, err = client.Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, nodePatchJson, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("error applying patch to node: %v", err)
	}

	return nil
}
