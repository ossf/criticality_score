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


docker-targets = build/docker/enumerate-github
.PHONY: build/docker $(docker-targets)
build/docker: $(docker-targets)  ## Build all docker targets
build/docker/enumerate-github:
	DOCKER_BUILDKIT=1 docker build . -f cmd/enumerate_github/Dockerfile --tag $(IMAGE_NAME)-enumerate-github
