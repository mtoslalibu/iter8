#!/usr/bin/env bash
#
# Copyright 2019 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

if [ -z "${KUBECTL_VERSION}" ]
then
    KUBECTL_VERSION=v1.11.0
fi

if [ -z "${KUSTOMIZE_VERSION}" ]
then
    KUSTOMIZE_VERSION=1.0.10
fi

echo "installing kubectl"
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/

echo "installing kustomize"
curl -OL "https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_amd64"
sudo chmod +x kustomize_${KUSTOMIZE_VERSION}_linux_amd64
sudo mv kustomize_${KUSTOMIZE_VERSION}_linux_amd64 /usr/local/bin/kustomize
kustomize version

echo "installing idt"
curl -sL https://ibm.biz/idt-installer | bash

