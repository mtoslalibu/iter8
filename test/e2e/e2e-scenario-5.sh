#!/usr/bin/env bash

# Exit on error
#set -e

DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1; pwd -P )"
source "$DIR/library.sh"

# This scenario reuses the Istio Gateway and Virtual Service created in scenario 1

header "Scenario 5"

header "Create productpage-gateway gateway"
kubectl apply -n bookinfo-iter8 -f $DIR/../../doc/tutorials/istio/bookinfo/productpage-gateway.yaml
kubectl get gateway -n bookinfo-iter8

header "Generate workload"
# We are using nodeport of the Istio ingress gateway to access bookinfo app
IP='127.0.0.1'
PORT=`kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}'`
# Following uses the K8s service IP/port to access bookinfo app
echo "Bookinfo is accessed at $IP:$PORT"
curl -H "Host: productpage.deployment.com" -Is "http://$IP:$PORT/productpage"
watch -n 0.1 "curl -H \"Host: productpage.deployment.com\" -Is \"http://$IP:$PORT/productpage\"" >/dev/null 2>&1 &

header "Create Iter8 Experiment"
kubectl apply -n bookinfo-iter8 -f $DIR/../../doc/tutorials/istio/bookinfo/canary_productpage-v1_to_productpage-v2.yaml
kubectl get experiments -n bookinfo-iter8

header "Deploy canary version"
kubectl apply -n bookinfo-iter8 -f $DIR/../../doc/tutorials/istio/bookinfo/productpage-v2.yaml
sleep 1
kubectl wait --for=condition=ExperimentCompleted -n bookinfo-iter8 experiments.iter8.tools productpage-v2-rollout --timeout=540s
kubectl get experiments -n bookinfo-iter8
kubectl get vs -n bookinfo-iter8 -o yaml

header "Test results"
kubectl -n bookinfo-iter8 get experiments.iter8.tools productpage-v2-rollout -o yaml
conclusion=`kubectl -n bookinfo-iter8 get experiments.iter8.tools productpage-v2-rollout -o=jsonpath='{.status.assessment.conclusions[0]}'`
if [ "$conclusion" != "All success criteria were  met" ]; then
  echo "Experiment failed unexpectedly!"
  exit 1
fi
echo "Experiment succeeded as expected!"