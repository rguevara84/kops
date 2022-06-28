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

package cloudup

import (
	"fmt"
	"net/url"

	"github.com/blang/semver/v4"

	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/assets"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/util/pkg/architectures"
	"k8s.io/kops/util/pkg/hashing"
)

const (
	runcVersion         = "1.1.3"
	runcVersionUrlAmd64 = "https://github.com/opencontainers/runc/releases/download/v%s/runc.amd64"
	runcVersionUrlArm64 = "https://github.com/opencontainers/runc/releases/download/v%s/runc.arm64"
)

func findRuncAsset(c *kops.Cluster, assetBuilder *assets.AssetBuilder, arch architectures.Architecture) (*url.URL, *hashing.Hash, error) {
	if c.Spec.Containerd == nil || c.Spec.Containerd.Version == nil {
		return nil, nil, fmt.Errorf("unable to find containerd version, used to determine runc version")
	}

	containerdVersion, err := semver.ParseTolerant(fi.StringValue(c.Spec.Containerd.Version))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse version string: %q", fi.StringValue(c.Spec.Containerd.Version))
	}
	// The a compatible runc binary is bundled with containerd builds < v1.6.0
	// https://github.com/containerd/containerd/issues/6541
	if containerdVersion.LT(semver.MustParse("1.6.0")) {
		return nil, nil, nil
	}

	version := runcVersion
	assetUrl, assetHash, err := findRuncVersionUrlHash(arch, version)
	if err != nil {
		return nil, nil, err
	}

	return findAssetsUrlHash(assetBuilder, assetUrl, assetHash)
}

func findRuncVersionUrlHash(arch architectures.Architecture, version string) (u string, h string, e error) {
	var runcAssetUrl, runcAssetHash string

	if findAllRuncHashesAmd64()[version] != "" {
		var err error
		runcAssetUrl, err = findRuncVersionUrl(arch, version)
		if err != nil {
			return "", "", err
		}
		runcAssetHash, err = findRuncVersionHash(arch, version)
		if err != nil {
			return "", "", err
		}
	} else {
		return "", "", fmt.Errorf("unknown url and hash for runc version: %s - %s", arch, version)
	}

	return runcAssetUrl, runcAssetHash, nil
}

func findRuncVersionUrl(arch architectures.Architecture, version string) (string, error) {
	sv, err := semver.ParseTolerant(version)
	if err != nil {
		return "", fmt.Errorf("unable to parse version string: %q", version)
	}
	if sv.LT(semver.MustParse("1.1.0")) {
		return "", fmt.Errorf("unsupported runc version: %q", version)
	}

	var u string
	switch arch {
	case architectures.ArchitectureAmd64:
		u = fmt.Sprintf(runcVersionUrlAmd64, version)
	case architectures.ArchitectureArm64:
		u = fmt.Sprintf(runcVersionUrlArm64, version)
	default:
		return "", fmt.Errorf("unknown arch: %q", arch)
	}

	if u == "" {
		return "", fmt.Errorf("unknown url for runc version: %s - %s", arch, version)
	}

	return u, nil
}

func findRuncVersionHash(arch architectures.Architecture, version string) (string, error) {
	sv, err := semver.ParseTolerant(version)
	if err != nil {
		return "", fmt.Errorf("unable to parse version string: %q", version)
	}
	if sv.LT(semver.MustParse("1.1.0")) {
		return "", fmt.Errorf("unsupported runc version: %q", version)
	}

	var h string
	switch arch {
	case architectures.ArchitectureAmd64:
		h = findAllRuncHashesAmd64()[version]
	case architectures.ArchitectureArm64:
		h = findAllRuncHashesArm64()[version]
	default:
		return "", fmt.Errorf("unknown arch: %q", arch)
	}

	if h == "" {
		return "", fmt.Errorf("unknown hash for runc version: %s - %s", arch, version)
	}

	return h, nil
}

func findAllRuncHashesAmd64() map[string]string {
	hashes := map[string]string{
		"1.1.0": "ab1c67fbcbdddbe481e48a55cf0ef9a86b38b166b5079e0010737fd87d7454bb",
		"1.1.1": "5798c85d2c8b6942247ab8d6830ef362924cd72a8e236e77430c3ab1be15f080",
		"1.1.2": "e0436dfc5d26ca88f00e84cbdab5801dd9829b1e5ded05dcfc162ce5718c32ce",
		"1.1.3": "6e8b24be90fffce6b025d254846da9d2ca6d65125f9139b6354bab0272253d01",
	}

	return hashes
}

func findAllRuncHashesArm64() map[string]string {
	hashes := map[string]string{
		"1.1.0": "9ec8e68feabc4e7083a4cfa45ebe4d529467391e0b03ee7de7ddda5770b05e68",
		"1.1.1": "20c436a736547309371c7ac2a335f5fe5a42b450120e497d09c8dc3902c28444",
		"1.1.2": "6ebd968d46d00a3886e9a0cae2e0a7b399e110cf5d7b26e63ce23c1d81ea10ef",
		"1.1.3": "00c9ad161a77a01d9dcbd25b1d76fa9822e57d8e4abf26ba8907c98f6bcfcd0f",
	}

	return hashes
}
