#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

. "$(dirname "${BASH_SOURCE[0]}")/common.sh"

files_need_boilerplate=()
while IFS='' read -r line; do files_need_boilerplate+=("$line"); done < <("${KOPS_ROOT}/hack/boilerplate/boilerplate.py" "$@")

if [[ -z ${files_need_boilerplate+x} ]]; then
    exit
fi

if [[ ${#files_need_boilerplate[@]} -gt 0 ]]; then
  for file in "${files_need_boilerplate[@]}"; do
    echo "FAIL: Boilerplate header is wrong for: ${file}"
  done
  echo "FAIL: Please execute ./hack/update-header.sh"
  exit 1
fi
