# Build the manager binary
FROM golang:1.21 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
ENV GOPROXY=https://goproxy.cn
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/* cmd/*
COPY api/ api/
COPY internal/ internal/
COPY Makefile Makefile

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
# RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o greatsql-operator cmd/main.go
RUN make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# registry.cn-chengdu.aliyuncs.com/gcr-distroless/static China mirror of distroless image.
# FROM gcr.io/distroless/static:nonroot
FROM registry.cn-chengdu.aliyuncs.com/gcr-distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/greatsql-operator .
USER 65532:65532

ENTRYPOINT ["/greatsql-operator"]
