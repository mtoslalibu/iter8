#!/usr/bin/env bash

echo fetching reviews domain
DOMAIN=$(kubectl get ksvc stock-experiment-example -o=jsonpath='{.status.domain}')

echo configuring ingress
kubectl apply -f - << EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: stock-experiment-example
  namespace: istio-system
  labels:
    app.kubernetes.io/name: stock-experiment-example
spec:
  rules:
  - host: ${DOMAIN}
    http:
      paths:
        - path: /
          backend:
            serviceName: 'istio-ingressgateway'
            servicePort: 80
EOF

echo $DOMAIN
echo