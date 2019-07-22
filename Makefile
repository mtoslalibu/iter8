
# Image URL to use all building/pushing image targets
IMG ?= iter8-controller:latest

all: manager

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/iter8.tools/iter8-controller/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kubectl apply -f config/rbac
	kubectl apply -f config/default

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
	echo '  namespace: iter8' >> ./config/rbac/manager_role_binding.yaml
	sed -i'' -e '12s@name:.*@name: controller-manager@' ./config/rbac/manager_role_binding.yaml
	rm -f ./config/rbac/manager_role_binding.yaml-e
	./hack/crd_fix.sh

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build:
	docker build . -t ${IMG}
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager.yaml
	rm -f ./config/default/manager.yaml-e

# Push the docker image
docker-push:
	docker push ${IMG}
