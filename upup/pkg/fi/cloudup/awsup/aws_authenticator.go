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

package awsup

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"k8s.io/kops/pkg/bootstrap"
)

const AWSAuthenticationTokenPrefix = "x-aws-sts "

type awsAuthenticator struct {
	sts *sts.STS
}

var _ bootstrap.Authenticator = &awsAuthenticator{}

// RegionFromMetadata returns the current region from the aws metdata
func RegionFromMetadata(ctx context.Context) (string, error) {
	config := aws.NewConfig()
	config = config.WithCredentialsChainVerboseErrors(true)

	s, err := session.NewSession(config)
	if err != nil {
		return "", err
	}
	metadata := ec2metadata.New(s, config)

	region, err := metadata.RegionWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get region from ec2 metadata: %w", err)
	}
	return region, nil
}

func NewAWSAuthenticator(region string) (bootstrap.Authenticator, error) {
	config := aws.NewConfig().
		WithCredentialsChainVerboseErrors(true).
		WithRegion(region).
		WithSTSRegionalEndpoint(endpoints.RegionalSTSEndpoint)
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	return &awsAuthenticator{
		sts: sts.New(sess, config),
	}, nil
}

func (a awsAuthenticator) CreateToken(body []byte) (string, error) {
	sha := sha256.Sum256(body)

	stsRequest, _ := a.sts.GetCallerIdentityRequest(nil)

	// Ensure the signature is only valid for this particular body content.
	stsRequest.HTTPRequest.Header.Add("X-Kops-Request-SHA", base64.RawStdEncoding.EncodeToString(sha[:]))

	if err := stsRequest.Sign(); err != nil {
		return "", err
	}

	headers, _ := json.Marshal(stsRequest.HTTPRequest.Header)
	return AWSAuthenticationTokenPrefix + base64.StdEncoding.EncodeToString(headers), nil
}
