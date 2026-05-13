# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.3@sha256:313faae491b410a35402c05d35e7518ae99103d957308e940e1ae2cfa0aac29b AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /pod-tls-sidecar main.go

FROM ubuntu
COPY --from=builder /pod-tls-sidecar /usr/bin/pod-tls-sidecar
ENTRYPOINT ["/usr/bin/pod-tls-sidecar"]
