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

package channels

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"k8s.io/klog/v2"
)

// Apply calls kubectl apply to apply the manifest.
// We will likely in future change this to create things directly (or more likely embed this logic into kubectl itself)
func Apply(data []byte) error {
	// We copy the manifest to a temp file because it is likely e.g. an s3 URL, which kubectl can't read
	tmpDir, err := os.MkdirTemp("", "channel")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %v", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			klog.Warningf("error deleting temp dir %q: %v", tmpDir, err)
		}
	}()

	localManifestFile := path.Join(tmpDir, "manifest.yaml")
	if err := os.WriteFile(localManifestFile, data, 0o600); err != nil {
		return fmt.Errorf("error writing temp file: %v", err)
	}
	{
		_, err := execKubectl("apply", "-f", localManifestFile, "--server-side", "--force-conflicts", "--field-manager=kops")
		if err != nil {
			klog.Errorf("failed to apply the manifest: %v", err)
		}
	}
	{
		_, err := execKubectl("replace", "-f", localManifestFile, "--field-manager=kops")
		if err != nil {
			return fmt.Errorf("failed to replace manifest: %w", err)
		}
	}

	// Remove this one. Just to show that apply works properly after replace
	{
		_, err := execKubectl("apply", "-f", localManifestFile, "--server-side", "--force-conflicts", "--field-manager=kops")
		if err != nil {
			return fmt.Errorf("failed to apply the manifest: %w", err)
		}
	}

	return nil
}

func execKubectl(args ...string) (string, error) {
	kubectlPath := "kubectl" // Assume in PATH
	cmd := exec.Command(kubectlPath, args...)
	env := os.Environ()
	cmd.Env = env

	human := strings.Join(cmd.Args, " ")
	klog.V(2).Infof("Running command: %s", human)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Infof("error running %s", human)
		klog.Info(string(output))
		return string(output), fmt.Errorf("error running kubectl: %v", err)
	}

	return string(output), err
}
