FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

ARG DOCTL_VERSION=1.135.0
RUN curl -sSL https://github.com/digitalocean/doctl/releases/download/v${DOCTL_VERSION}/doctl-${DOCTL_VERSION}-linux-amd64.tar.gz | \
    tar -xz -C /usr/local/bin
