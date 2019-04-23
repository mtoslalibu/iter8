# Stock ticker Knative sample

## Prerequisites

* Kubernetes 1.11+
* Knative 0.5.0+
* kubectl 1.11+
* [Ko](https://github.com/google/ko). Make sure to set `$KO_DOCKER_REPO`

## Deploying

1. deploy stock ticker Knative service:

```sh
go get ./...
ko apply -f stock-svc.yaml
```

2. (optional) If you installed Knative manually (not as an add-on), create an ingress:

```sh
./configure-ingress.sh
```

## Verifying

1. Get the stock service endpoint and curl it:

```sh
DOMAIN=$(kubectl get ksvc stock-canary-example -o=jsonpath='{.status.domain}')
curl $DOMAIN

Welcome to the stock app!
```

2. In a separate terminal, [run the controller locally](../../README.md#run-the-controller-locally)

3. Configure the canary:

```sh
kubectl apply -f canary.yaml
```

Wait a bit and get the service:

```sh
kubectl get ksvc stock-canary-example -oyaml
```

You should see `runLatest` has been replaced by `release`.

The canary condition `Ready` is `True` as the service traffic is 100% directed to the current revision.

```sh
kubectl get canary stock-canary-example -oyaml
```

4. Create a new stock app revision

```sh
kubectl apply -f stock-share-svc.yaml
```

The canary controller automatically promotes the latest revision to candidate.
It also automatically starts shifting traffic from current to canditate by a 2% increment every 1 minute, until
reaching a max of 50%.

Observe the traffic shifting by invoking the service multiple times:

```sh
Welcome to the stock app!
Welcome to the stock app!
Welcome to the share app!
```


## Cleaning up

```sh
kubectl delete all,serviceentry -l 'app.kubernetes.io/name=stock-canary-example'
```

## Common issues

If you get `error processing import paths in "stock-svc.yaml": unsupported status code 401; body:`

you need to login to docker.
