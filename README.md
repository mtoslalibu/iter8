# iter8-controller

## Getting Started

### Prerequisites

* Kubernetes 1.11+

### Run the controller locally

1. Clone this repository under `$GOPATH/src/github.ibm.com/istio-research/iter8-controller`
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
