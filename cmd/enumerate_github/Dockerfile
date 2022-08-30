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

FROM golang@sha256:684d791f8fd1b28485ee32239d92cde7099d97c88c5ed2e955f65f4adcf35d0f AS base
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

FROM base AS enumerate_github
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 go build ./cmd/enumerate_github

FROM gcr.io/distroless/base:nonroot@sha256:533c15ef2acb1d3b1cd4e58d8aa2740900cae8f579243a53c53a6e28bcac0684
COPY --from=enumerate_github /src/enumerate_github ./enumerate_github
ENTRYPOINT ["./enumerate_github"]
