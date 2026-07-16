# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.2@sha256:b54cbf583d390341599d7bcbc062425c081105cc5ef6d170ced98ef9d047c716 AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /pod-tls-sidecar main.go

FROM ubuntu
COPY --from=builder /pod-tls-sidecar /usr/bin/pod-tls-sidecar
ENTRYPOINT ["/usr/bin/pod-tls-sidecar"]
