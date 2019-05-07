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

ROOT=$(dirname $0)
source $ROOT/../scripts/library.sh

function cleanup() {
  if [ -n "$WATCH_NAMESPACE" ]
  then
    header "deleting namespace $WATCH_NAMESPACE"
    kubectl delete ns $WATCH_NAMESPACE
  fi
}

function traperr() {
  echo "ERROR: ${BASH_SOURCE[1]} at about ${BASH_LINENO[0]}"
  cleanup
}

set -o errtrace
trap traperr ERR

parse_flags $*

if [ -z "$SKIP_SETUP" ]
then
    setup_knative
fi

WATCH_NAMESPACE=$(random_namespace)
header "creating namespace $WATCH_NAMESPACE"
kubectl create ns $WATCH_NAMESPACE

header "install iter8"
make install

header "run iter8 controller locally"
go run ./cmd/manager/main.go &
CONTROLLER_PID=$!
echo "controller started $CONTROLLER_PID"

kill $CONTROLLER_PID


cleanup



