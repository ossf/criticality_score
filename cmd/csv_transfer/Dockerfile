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

FROM golang@sha256:122f3484f844467ebe0674cf57272e61981770eb0bc7d316d1f0be281a88229f AS base
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

FROM base AS csv_transfer
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 go build ./cmd/csv_transfer

FROM gcr.io/distroless/base:nonroot@sha256:533c15ef2acb1d3b1cd4e58d8aa2740900cae8f579243a53c53a6e28bcac0684
COPY --from=csv_transfer /src/csv_transfer ./csv_transfer
ENTRYPOINT ["./csv_transfer"]
