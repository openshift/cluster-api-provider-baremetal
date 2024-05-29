
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
GOPATH=$(shell go env GOPATH)

BIN_DIR := bin
MOCKGEN := $(BIN_DIR)/mockgen
SETUP_ENVTEST := $(BIN_DIR)/setup-envtest
ENVTEST_K8S_VERSION := 1.29.x
ENVTEST_OS := linux
ARCH ?= amd64

$(MOCKGEN): # Build mockgen
	go build -tags=tools -o $(BIN_DIR)/mockgen github.com/golang/mock/mockgen

$(SETUP_ENVTEST):
	go build -tags=tools -o $(BIN_DIR)/setup-envtest sigs.k8s.io/controller-runtime/tools/setup-envtest

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
unit-test: $(SETUP_ENVTEST)
	$(shell $(SETUP_ENVTEST) use -p env --os $(ENVTEST_OS) --arch $(ARCH) $(ENVTEST_K8S_VERSION)) && \
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
