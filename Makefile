
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
GOPATH=$(shell go env GOPATH)

BIN_DIR := bin
MOCKGEN := $(BIN_DIR)/mockgen
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION := 1.34.1

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
ENVTEST = go run ${PROJECT_DIR}/vendor/sigs.k8s.io/controller-runtime/tools/setup-envtest

$(MOCKGEN): # Build mockgen
	go build -tags=tools -o $(BIN_DIR)/mockgen github.com/golang/mock/mockgen

.PHONY: build
build:
	@mkdir -p bin
	go build --mod=vendor -o bin/machine-controller-manager ./cmd/manager

all: test manager

# Run tests
test: generate fmt vet unit

.PHONY: unit
unit: unit-test

.PHONY: unit-test
unit-test:
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(PROJECT_DIR)/bin --index https://raw.githubusercontent.com/openshift/api/master/envtest-releases.yaml)" \
		go test ./pkg/... ./cmd/... -coverprofile cover.out

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
.PHONY: install
install:
	kubectl apply -f vendor/github.com/openshift/machine-api-operator/install
	kustomize build config | kubectl apply -f -

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
.PHONY: generate
generate: $(MOCKGEN)
	go generate ./pkg/... ./cmd/...

	$(MOCKGEN) \
	  -destination=./pkg/baremetal/mocks/zz_generated.metal3remediation_manager.go \
	  -source=./pkg/baremetal/metal3remediation_manager.go \
		-package=baremetal_mocks \
		-copyright_file=./hack/boilerplate.go.txt \
		RemediationManagerInterface

	$(MOCKGEN) \
	  -destination=./pkg/baremetal/mocks/zz_generated.manager_factory.go \
	  -source=./pkg/baremetal/manager_factory.go \
		-package=baremetal_mocks \
		-copyright_file=./hack/boilerplate.go.txt \
		ManagerFactoryInterface

.PHONY: generate-check
generate-check:
	./hack/generate.sh

# Build the docker image
.PHONY: docker-build
docker-build: test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: metal3-crds
metal3-crds:
	./hack/fetch-metal3-crds.sh
