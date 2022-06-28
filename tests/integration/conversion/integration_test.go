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

package main

import (
	"bytes"
	"os"
	"path"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kops/pkg/apis/kops/v1alpha2"
	"k8s.io/kops/pkg/apis/kops/v1alpha3"
	"k8s.io/kops/pkg/diff"
	"k8s.io/kops/pkg/kopscodecs"
	"k8s.io/kops/util/pkg/text"
)

// TestConversionMinimal runs the test on a minimum configuration, similar to kops create cluster minimal.example.com --zones us-west-1a
func TestConversionMinimal(t *testing.T) {
	runTest(t, "minimal", "legacy-v1alpha2", "v1alpha2")
	runTest(t, "minimal", "v1alpha2", "v1alpha3")
	runTest(t, "minimal", "v1alpha3", "v1alpha2")
}

func TestConversionAWS(t *testing.T) {
	runTest(t, "aws", "v1alpha2", "v1alpha3")
	runTest(t, "aws", "v1alpha3", "v1alpha2")
}

func TestConversionAzure(t *testing.T) {
	runTest(t, "azure", "v1alpha2", "v1alpha3")
	runTest(t, "azure", "v1alpha3", "v1alpha2")
}

func TestConversionCanal(t *testing.T) {
	runTest(t, "canal", "v1alpha2", "v1alpha3")
	runTest(t, "canal", "v1alpha3", "v1alpha2")
}

func TestConversionCilium(t *testing.T) {
	runTest(t, "cilium", "v1alpha2", "v1alpha3")
	runTest(t, "cilium", "v1alpha3", "v1alpha2")
}

func TestConversionDO(t *testing.T) {
	runTest(t, "do", "v1alpha2", "v1alpha3")
	runTest(t, "do", "v1alpha3", "v1alpha2")
}

func TestConversionGCE(t *testing.T) {
	runTest(t, "gce", "v1alpha2", "v1alpha3")
	runTest(t, "gce", "v1alpha3", "v1alpha2")
}

func TestConversionOpenstack(t *testing.T) {
	runTest(t, "openstack", "v1alpha2", "v1alpha3")
	runTest(t, "openstack", "v1alpha3", "v1alpha2")
}

func runTest(t *testing.T, srcDir string, fromVersion string, toVersion string) {
	t.Run(fromVersion+"-"+toVersion, func(t *testing.T) {
		sourcePath := path.Join(srcDir, fromVersion+".yaml")
		sourceBytes, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("unexpected error reading sourcePath %q: %v", sourcePath, err)
		}

		expectedPath := path.Join(srcDir, toVersion+".yaml")
		expectedBytes, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("unexpected error reading expectedPath %q: %v", expectedPath, err)
		}

		yaml, ok := runtime.SerializerInfoForMediaType(kopscodecs.Codecs.SupportedMediaTypes(), "application/yaml")
		if !ok {
			t.Fatalf("no YAML serializer registered")
		}
		var encoder runtime.Encoder

		switch toVersion {
		case "v1alpha2":
			encoder = kopscodecs.Codecs.EncoderForVersion(yaml.Serializer, v1alpha2.SchemeGroupVersion)
		case "v1alpha3":
			encoder = kopscodecs.Codecs.EncoderForVersion(yaml.Serializer, v1alpha3.SchemeGroupVersion)

		default:
			t.Fatalf("unknown version %q", toVersion)
		}

		var actual []string

		sections := text.SplitContentToSections(sourceBytes)
		for _, s := range sections {
			o, gvk, err := kopscodecs.Decode([]byte(s), nil)
			if err != nil {
				t.Fatalf("error parsing file %q: %v", sourcePath, err)
			}

			expectVersion := strings.TrimPrefix(fromVersion, "legacy-")
			if expectVersion == "v1alpha0" {
				// Our version before we had v1alpha1
				expectVersion = "v1alpha1"
			}
			if gvk.Version != expectVersion {
				t.Fatalf("unexpected version: %q vs %q", gvk.Version, expectVersion)
			}

			var b bytes.Buffer
			if err := encoder.Encode(o, &b); err != nil {
				t.Fatalf("error encoding object: %v", err)
			}

			actual = append(actual, b.String())
		}

		actualString := strings.TrimSpace(strings.Join(actual, "\n---\n\n"))
		expectedString := strings.TrimSpace(string(expectedBytes))

		actualString = strings.Replace(actualString, "\r", "", -1)
		expectedString = strings.Replace(expectedString, "\r", "", -1)

		if actualString != expectedString {
			diffString := diff.FormatDiff(expectedString, actualString)
			t.Logf("diff:\n%s\n", diffString)

			t.Fatalf("%s->%s converted output differed from expected", fromVersion, toVersion)
		}
	})
}
