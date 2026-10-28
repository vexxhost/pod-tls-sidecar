# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /pod-tls-sidecar main.go

FROM ubuntu
COPY --from=builder /pod-tls-sidecar /usr/bin/pod-tls-sidecar
ENTRYPOINT ["/usr/bin/pod-tls-sidecar"]
