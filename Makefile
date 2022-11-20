# Copyright 2022 Criticality Score Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
IMAGE_NAME = criticality-score
GOLANGCI_LINT := golangci-lint

default: help

.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; \
			printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9\/-]+:.*?##/ \
			{ printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } \
			/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: test
test:  ## Run all tests
	go test -race -covermode=atomic -coverprofile=unit-coverage.out './...'

.PHONY: lint
$(GOLANGCI_LINT): install/deps
lint:  ## Run linter
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -c .golangci.yml

docker-targets = build/docker/enumerate-github build/docker/criticality-score build/docker/collect-signals build/docker/csv-transfer
.PHONY: build/docker $(docker-targets)
build/docker: $(docker-targets)  ## Build all docker targets
build/docker/collect-signals:
	DOCKER_BUILDKIT=1 docker build . -f cmd/collect_signals/Dockerfile --tag $(IMAGE_NAME)-collect-signals
build/docker/criticality-score:
	DOCKER_BUILDKIT=1 docker build . -f cmd/criticality_score/Dockerfile --tag $(IMAGE_NAME)-cli
build/docker/enumerate-github:
	DOCKER_BUILDKIT=1 docker build . -f cmd/enumerate_github/Dockerfile --tag $(IMAGE_NAME)-enumerate-github
build/docker/csv-transfer:
	DOCKER_BUILDKIT=1 docker build . -f cmd/csv_transfer/Dockerfile --tag $(IMAGE_NAME)-csv-transfer

.PHONY: install/deps
install/deps:  ## Installs all dependencies during development and building
	@echo Installing tools from tools/tools.go
	@cd tools; cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %
