# Resilience Operator Makefile

# Image URL to use all building/pushing image targets
IMG ?= resilience-operator:latest
ENVTEST_K8S_VERSION = 1.31.0

# Get the currently used golang install path
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN = $(shell go env GOPATH)/bin
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CRD manifests.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code (DeepCopy).
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

##@ Testing

.PHONY: test
test: manifests generate fmt vet envtest ## Run all tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: ## Run unit tests.
	go test ./tests/unit/... -v -coverprofile cover-unit.out

.PHONY: test-integration
test-integration: envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./tests/integration/... -v

.PHONY: test-property
test-property: ## Run property-based tests.
	go test ./tests/property/... -v -count=1

.PHONY: test-e2e
test-e2e: ## Run e2e tests (requires cluster).
	go test ./tests/e2e/... -v -count=1

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run against cluster.
	go run ./cmd/main.go

##@ Docker

.PHONY: docker-build
docker-build: ## Build docker image.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push ${IMG}

.PHONY: docker-buildx
docker-buildx: ## Build and push multi-arch image.
	docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG} --push .

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy to cluster.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy from cluster.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found -f -

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint Helm chart.
	helm lint ../../../deploy/kubernetes/service-mesh/helm/resilience-operator

.PHONY: helm-template
helm-template: ## Template Helm chart.
	helm template resilience-operator ../../../deploy/kubernetes/service-mesh/helm/resilience-operator

##@ Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
KUSTOMIZE ?= $(LOCALBIN)/kustomize
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@latest

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
