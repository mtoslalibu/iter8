
#!/usr/bin/env bash
#
# Copyright 2018 The Knative Authors
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

# This is a collection of useful bash functions and constants, intended
# to be used in test scripts and the like. It doesn't do anything when
# called from command line.


function setup_knative() {
  if [ -z "$IC_API_ENDPOINT" ]
  then
    install_knative
  else
    configure_cluster
  fi
}

function configure_cluster() {
  header "setting KUBECONFIG"

  ibmcloud login -a "$IC_API_ENDPOINT" --apikey "$IC_APIKEY" -r $IC_REGION
  ibmcloud ks region-set "$IC_REGION"
  $(ibmcloud ks cluster-config "$CLUSTER_NAME" --export -s)
}

function install_knative() {
  header "installing istio"
  kubectl apply --filename https://raw.githubusercontent.com/knative/serving/v0.5.2/third_party/istio-1.0.7/istio-crds.yaml
  curl -L https://raw.githubusercontent.com/knative/serving/v0.5.2/third_party/istio-1.0.7/istio.yaml \
    | sed 's/LoadBalancer/NodePort/' \
    | kubectl apply --filename -

  # Label the default namespace with istio-injection=enabled.
  kubectl label namespace default istio-injection=enabled --overwrite

  wait_until_pods_running "istio-system"

  header "installing serving and monitoring CRDs"
  kubectl apply --selector knative.dev/crd-install=true \
   -f https://github.com/knative/serving/releases/download/v0.5.2/serving.yaml \
   -f https://raw.githubusercontent.com/knative/serving/v0.5.2/third_party/config/build/clusterrole.yaml

   header "installing serving and monitoring controllers"
   kubectl apply -f https://github.com/knative/serving/releases/download/v0.5.2/serving.yaml \
   -f https://raw.githubusercontent.com/knative/serving/v0.5.2/third_party/config/build/clusterrole.yaml

   wait_until_pods_running "knative-serving"
}

# Simple header for logging purposes.
function header() {
  local upper="$(echo $1 | tr a-z A-Z)"
  make_banner "=" "${upper}"
}

# Display a box banner.
# Parameters: $1 - character to use for the box.
#             $2 - banner message.
function make_banner() {
    local msg="$1$1$1$1 $2 $1$1$1$1"
    local border="${msg//[-0-9A-Za-z _.,\/()]/$1}"
    echo -e "${border}\n${msg}\n${border}"
}


# Waits until all pods are running in the given namespace.
# Parameters: $1 - namespace.
function wait_until_pods_running() {
  echo -n "Waiting until all pods in namespace $1 are up"
  for i in {1..150}; do  # timeout after 5 minutes
    local pods="$(kubectl get pods --no-headers -n $1 2>/dev/null)"
    # All pods must be running
    local not_running=$(echo "${pods}" | grep -v Running | grep -v Completed | wc -l)
    if [[ -n "${pods}" && ${not_running} -eq 0 ]]; then
      local all_ready=1
      while read pod ; do
        local status=(`echo -n ${pod} | cut -f2 -d' ' | tr '/' ' '`)
        # All containers must be ready
        [[ -z ${status[0]} ]] && all_ready=0 && break
        [[ -z ${status[1]} ]] && all_ready=0 && break
        [[ ${status[0]} -lt 1 ]] && all_ready=0 && break
        [[ ${status[1]} -lt 1 ]] && all_ready=0 && break
        [[ ${status[0]} -ne ${status[1]} ]] && all_ready=0 && break
      done <<< $(echo "${pods}" | grep -v Completed)
      if (( all_ready )); then
        echo -e "\nAll pods are up:\n${pods}"
        return 0
      fi
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for pods to come up\n${pods}"
  return 1
}

function parse_flags() {
  while (( "$#" )); do
    case "$1" in
      -s|--skip-setup)
        SKIP_SETUP=1
        shift 1
        ;;
      --) # end argument parsing
        shift
        break
        ;;
      -*|--*=) # unsupported flags
        echo "Error: Unsupported flag $1" >&2
        exit 1
        ;;
    esac
  done
}

function random_namespace() {
  ns="iter8-testing-$(cat /dev/urandom | env LC_CTYPE=C tr -dc 'a-z0-9' | fold -w 6 | head -n 1)"
  echo $ns
}