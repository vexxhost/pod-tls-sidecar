# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.4@sha256:5e8afd493a7124a8586e134d9b532731161d4b9becf9dee5a5956b107426f93d AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /pod-tls-sidecar main.go

FROM ubuntu
COPY --from=builder /pod-tls-sidecar /usr/bin/pod-tls-sidecar
ENTRYPOINT ["/usr/bin/pod-tls-sidecar"]
