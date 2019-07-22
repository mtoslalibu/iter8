# iter8-controller

[![Build Status](https://travis.ibm.com/istio-research/iter8-controller.svg?token=bc6xtkRixk96zbXuAu7U&branch=master)](https://travis.ibm.com/istio-research/iter8-controller) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Getting Started

### Prerequisites

* Kubernetes 1.11+

### Run the controller locally

1. Clone this repository under `$GOPATH/src/github.com/iter8.tools/iter8-controller`
2. Install CRDs into the cluste:

```sh
make install
```

3. Run the controller locally:

```sh
make run
```

4. In a new terminal - create an instance and expect the Controller to pick it up

```sh
kubectl apply -f config/samples/iter8_v1alpha1_canary.yaml
```

## Run the Demo

### Stage 1 of the demo

1. `kubectl apply -f samples/bookinfo/bookinfowithdelay.yaml`
2. `kubectl apply -f samples/bookinfo/bookinfo-gateway.yaml`
3. `kubectl apply -f canaryv1v2.yaml`
4. `kubectl apply -f reviews_v2.yaml`

### Stage 2 of the demo

5. `kubectl apply -f canaryv2v3.yaml`
6. `kubectl apply -f reviews_v3.yaml`
