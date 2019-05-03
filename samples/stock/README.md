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

3. Create an LoadBalancer service for Knative Prometheus

```sh
kubectl apply -f prometheus-lb.yaml
ITER8_ANALYTICS_METRICS_BACKEND_URL=http://$(k get services prometheus-system-np-lb -o=jsonpath='{.status.loadBalancer.ingress[0].ip}' -n knative-monitoring)
```

## Verifying

1. Start the [analytic service locally](https://github.ibm.com/istio-research/iter8/tree/master/scripts)

Make sure `$ITER8_ANALYTICS_METRICS_BACKEND_URL` is set (see above)

2. Get the stock service endpoint and curl it:

```sh
DOMAIN=$(kubectl get ksvc stock-canary-example -o=jsonpath='{.status.domain}')
curl $DOMAIN

Welcome to the stock app!
```

3. In a separate terminal, [run the controller locally](../../README.md#run-the-controller-locally)

4. Configure the manual canary:

```sh
kubectl apply -f canary-manual.yaml
```

Wait a bit and get the service:

```sh
kubectl get ksvc stock-canary-example -oyaml
```

You should see `runLatest` has been replaced by `release`.

The canary condition `Ready` is `True` as the service traffic is 100% directed to the current revision.

```sh
kubectl get canary.iter8.ibm.com stock-canary-example -oyaml
```

5. Create a new stock app revision

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
