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

package components

import (
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/loader"
)

// DockerOptionsBuilder adds options for docker to the model
type DockerOptionsBuilder struct {
	*OptionsContext
}

var _ loader.OptionsBuilder = &DockerOptionsBuilder{}

// BuildOptions is responsible for filling in the default setting for docker daemon
func (b *DockerOptionsBuilder) BuildOptions(o interface{}) error {
	clusterSpec := o.(*kops.ClusterSpec)

	if clusterSpec.Docker == nil {
		clusterSpec.Docker = &kops.DockerConfig{}
	}

	docker := clusterSpec.Docker

	// Container runtime is not Docker, should not install
	if clusterSpec.ContainerRuntime != "docker" {
		docker.SkipInstall = true
		return nil
	}

	// Set the Docker version for known Kubernetes versions
	if fi.StringValue(clusterSpec.Docker.Version) == "" {
		if b.IsKubernetesGTE("1.21") {
			docker.Version = fi.String("20.10.17")
		} else {
			docker.Version = fi.String("19.03.15")
		}
	}

	if len(clusterSpec.Docker.LogOpt) == 0 && clusterSpec.Docker.LogDriver == nil {
		// Use built-in docker logging, if not configured otherwise (by the user)
		logDriver := "json-file"
		clusterSpec.Docker.LogDriver = &logDriver
		clusterSpec.Docker.LogOpt = append(clusterSpec.Docker.LogOpt, "max-size=10m")
		clusterSpec.Docker.LogOpt = append(clusterSpec.Docker.LogOpt, "max-file=5")
	}

	docker.LogLevel = fi.String("info")
	docker.IPTables = fi.Bool(false)
	docker.IPMasq = fi.Bool(false)

	// Note the alternative syntax... with a comma nodeup will try each of the filesystems in turn
	// TODO(justinsb): The ContainerOS image now has docker configured to use overlay2 out-of-the-box
	// and it is an error to specify the flag twice.
	docker.Storage = fi.String("overlay2,overlay,aufs")

	// Set systemd as the default cgroup driver in docker from k8s 1.20.
	if b.IsKubernetesGTE("1.20") && getDockerCgroupDriver(docker.ExecOpt) == "" {
		docker.ExecOpt = append(docker.ExecOpt, "native.cgroupdriver=systemd")
	}

	return nil
}

// checks if cgroup-driver is configured or not for docker or not.
func getDockerCgroupDriver(execOpts []string) string {
	for _, value := range execOpts {
		if value == "native.cgroupdriver=systemd" {
			return "systemd"
		} else if value == "native.cgroupdriver=cgroupfs" {
			return "cgroupfs"
		}
	}

	return ""
}
