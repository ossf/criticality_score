FROM golang@sha256:684d791f8fd1b28485ee32239d92cde7099d97c88c5ed2e955f65f4adcf35d0f AS base
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* ./
RUN go mod download
COPY . ./

FROM base AS enumerate_github
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 go build ./cmd/enumerate_github

FROM gcr.io/distroless/base:nonroot@sha256:533c15ef2acb1d3b1cd4e58d8aa2740900cae8f579243a53c53a6e28bcac0684
COPY --from=enumerate_github /src/enumerate_github ./enumerate_github
ENTRYPOINT ["./enumerate_github"]
