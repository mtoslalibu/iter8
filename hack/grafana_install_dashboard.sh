#!/bin/bash

#set -x

verlte() {
  [ "$1" = "`echo -e "$1\n$2" | sort -V | head -n1`" ]
}

verlt() {
  [ "$1" = "$2" ] && return 1 || verlte $1 $2
}

autodetect() {

  ISTIO_VERSION=`kubectl -n istio-system get pods -o yaml | grep "image:" | grep proxy | head -n 1 | awk -F: '{print $3}'`
  KUBERNETES_VERSION=`kubectl version | grep "Server Version"`
  KUBERNETES_VERSION_MAJOR=`echo "$KUBERNETES_VERSION" | awk -F\" '{print $2}'`
  KUBERNETES_VERSION_MINOR=`echo "$KUBERNETES_VERSION" | awk -F\" '{print $4}'`
  KUBERNETES_VERSION="$KUBERNETES_VERSION_MAJOR.$KUBERNETES_VERSION_MINOR"

  if [ -z "$ISTIO_VERSION" ]; then
    echo "Cannot detect Istio version, aborting..."
    return
  elif [ -z "$KUBERNETES_VERSION" ]; then
    echo "Cannot detect Kubernetes version, aborting..."
    return
  fi

  echo "Istio version: $ISTIO_VERSION"
  echo "Kubernetes version: $KUBERNETES_VERSION"

  if verlt "$ISTIO_VERSION" "1.5"; then

    echo "Using Istio telemetry v1"

    if verlt "$KUBERNETES_VERSION" "1.16"; then
      echo "Using Prometheus queries for older Kubernetes (<v1.16) "
      DASHBOARD_DEFN="https://raw.githubusercontent.com/iter8-tools/iter8-controller/v0.2/config/grafana/istio-telemetry-v1.json"
    else
      echo "Using Prometheus queries for newer Kubernetes (>=v1.16)"
      DASHBOARD_DEFN="https://raw.githubusercontent.com/iter8-tools/iter8-controller/v0.2/config/grafana/istio-telemetry-v1-k8s-16.json"
    fi

  else
    echo "Using Istio telemetry v2"

    if verlt "$KUBERNETES_VERSION" "1.16"; then
      echo "Using Prometheus queries for older Kubernetes (<v1.16) "
      DASHBOARD_DEFN="https://raw.githubusercontent.com/iter8-tools/iter8-controller/v0.2/config/grafana/istio-telemetry-v2.json"
    else
      echo "Using Prometheus queries for newer Kubernetes (>=v1.16)"
      DASHBOARD_DEFN="https://raw.githubusercontent.com/iter8-tools/iter8-controller/v0.2/config/grafana/istio-telemetry-v2-k8s-16.json"
    fi
  fi
}

# Run auto detection code only if $DASHBOARD_DEFN is not defined
[ -z $DASHBOARD_DEFN ] && autodetect

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

DASHBOARD_UID=eXPEaNnZz
: "${GRAFANA_URL:=http://localhost:3000}"
: "${DASHBOARD_DEFN:=${DIR}/../config/grafana/istio.json}"

echo "      GRAFANA_URL=$GRAFANA_URL"
echo "    DASHBOARD_UID=$DASHBOARD_UID"
echo "   DASHBOARD_DEFN=$DASHBOARD_DEFN"

function get_config {
  if [[ "${DASHBOARD_DEFN}" == https://* ]] || \
     [[ "${DASHBOARD_DEFN}" == http://* ]]; then
    curl -s ${DASHBOARD_DEFN} | cat -
  else
    cat ${DASHBOARD_DEFN}
  fi
}

status=$(curl -Is --header 'Accept: application/json' $GRAFANA_URL/api/dashboards/uid/$DASHBOARD_UID 2>/dev/null | head -n 1 | cut -d$' ' -f2)

if [[ "$status" == "200" ]]; then
  echo "Canary Dashboard already defined in $GRAFANA_URL"
  # Could update by copying id, version from current dashboard
else
  echo "Defining canary dashboard on $GRAFANA_URL"
  echo "{ \"dashboard\": $(get_config) }" \
  | jq 'del(.dashboard.id) | del(.dashboard.version)' \
  | curl --request POST \
    --header 'Accept: application/json' \
    --header 'Content-Type: application/json' \
    $GRAFANA_URL/api/dashboards/db \
    --data @-
  echo ""
fi
