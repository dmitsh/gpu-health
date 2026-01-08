FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

ARG GO_VERSION=1.24.7
ARG TARGETARCH=arm64

RUN apt update && apt install -y wget ca-certificates \
 && wget https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz \
 && tar -C /usr/local -xzf go${GO_VERSION}.linux-${TARGETARCH}.tar.gz \
 && rm go${GO_VERSION}.linux-${TARGETARCH}.tar.gz

RUN apt install -y build-essential libnvidia-ml-dev

ENV PATH="/usr/local/go/bin:${PATH}"
