# Stock ticker Knative sample

## Prerequisites

* Kubernetes 1.11+
* Knative 0.5.0+
* kubectl 1.11+
* [Ko](https://github.com/google/ko). Make sure to set `$KO_DOCKER_REPO`
* [fortio](https://github.com/fortio/fortio) for load testing

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
ITER8_ANALYTICS_METRICS_BACKEND_URL=http://$(k get services prometheus-system-np-lb -o=jsonpath='{.status.loadBalancer.ingress[0].ip}' -n knative-monitoring):9090
```

## Verifying (locally)

1. Start the [analytic service locally](https://github.ibm.com/istio-research/iter8/tree/master/scripts)

Make sure `$ITER8_ANALYTICS_METRICS_BACKEND_URL` is set (see above)

2. Get the stock service endpoint and curl it:

```sh
DOMAIN=$(kubectl get ksvc stock-experiment-example -o=jsonpath='{.status.domain}')
curl $DOMAIN

Welcome to the stock app!
```

3. In a separate terminal, [run the controller locally](../../README.md#run-the-controller-locally)

4. Create a new stock app revision

```sh
kubectl apply -f stock-share-svc.yaml
```

5. Configure the experiment:

```sh
kubectl apply -f experiment.yaml
```

Wait a bit and get the service:

```sh
kubectl get ksvc stock-experiment-example -oyaml
```

You should see `runLatest` has been replaced by `release`.

The experiment condition `Ready` is `True` as the service traffic is 100% directed to the current revision.

```sh
kubectl get experiment.iter8-tools stock-experiment-example -oyaml
```

6. Generate some load

```sh
fortio load -t 10m -qps 100  http://$DOMAIN
```

The experiment controller automatically promotes the latest revision to candidate.

Observe the traffic shifting by invoking the service multiple times:

```sh
$ watch curl $DOMAIN
```

## Observing traffic with Graphana

1. Forward Knative monitoring Prometheus traffic to localhost

```sh
kubectl port-forward --namespace knative-monitoring $(kubectl get pods --namespace knative-monitoring --selector=app=grafana --output=jsonpath="{.items..metadata.name}") 3000
```

2. Open [http://localhost:3000](http://localhost:3000)

3. Open Knative Serving - Revision HTTP Requests dashboard

## Cleaning up

```sh
kubectl delete all,serviceentry -l 'app.kubernetes.io/name=stock-experiment-example'
```

## Common issues

If you get `error processing import paths in "stock-svc.yaml": unsupported status code 401; body:`

you need to login to docker.
