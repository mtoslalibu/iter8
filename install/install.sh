#!/bin/bash

#set -x

verlte() {
  [ "$1" = "`echo -e "$1\n$2" | sort -V | head -n1`" ]
}

verlt() {
  [ "$1" = "$2" ] && return 1 || verlte $1 $2
}

install() {

  ISTIO_VERSION=`kubectl -n istio-system get pods -o yaml | grep "image:" | grep proxy | head -n 1 | awk -F: '{print $3}'`

  if [ -z "$ISTIO_VERSION" ]; then
    echo "Cannot detect Istio version, aborting..."
    return
  fi

  echo "Istio version: $ISTIO_VERSION"

  if verlt "$ISTIO_VERSION" "1.5"; then
    echo "Using Istio telemetry v1"
    kubectl apply -f https://raw.githubusercontent.com/iter8-tools/iter8-controller/master/install/iter8-controller.yaml
  else
    echo "Using Istio telemetry v2"
	kubectl apply -f https://raw.githubusercontent.com/iter8-tools/iter8-controller/master/install/iter8-controller-telemetry-v2.yaml
  fi
  kubectl apply -f https://raw.githubusercontent.com/iter8-tools/iter8-analytics/master/install/kubernetes/iter8-analytics.yaml
}

install
