# Stock ticker Knative sample

## Prerequisites

* Kubernetes 1.11+
* Knative 0.5.0+
* kubectl 1.14+
* [Ko](https://github.com/google/ko). Make sure to set `$KO_DOCKER_REPO`

## Deploying

1. deploy stock ticker Knative service:

```sh
go get ./...
ko apply -f .
```

2. (optional) If you installed Knative manually (not as an add-on), create an ingress for reviews:

```sh
./configure-ingress.sh
```

## Cleaning up

```sh
kubectl delete -f .
```

## Common issues

If you get `error processing import paths in "stock-svc.yaml": unsupported status code 401; body:`

you need to login to docker.