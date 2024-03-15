ARG VERSION

FROM golang:1.22 AS build

COPY . /src
RUN cd /src && go build -ldflags="-X 'github.com/plumber-cd/argocd-applicationset-namespaces-generator-plugin/cmd/version.Version=$VERSION'" -o /bin/argocd-applicationset-namespaces-generator-plugin

FROM ubuntu:latest

RUN useradd -s /bin/bash -u 999 argocd
WORKDIR /home/argocd
USER argocd

ENTRYPOINT ["/usr/local/bin/argocd-applicationset-namespaces-generator-plugin", "server"]