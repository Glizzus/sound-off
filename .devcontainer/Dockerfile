FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

ARG DOCTL_VERSION=1.135.0
RUN curl -sSL https://github.com/digitalocean/doctl/releases/download/v${DOCTL_VERSION}/doctl-${DOCTL_VERSION}-linux-amd64.tar.gz | \
    tar -xz -C /usr/local/bin && \
    curl -ssL https://github.com/evilmartians/lefthook/releases/download/v1.12.3/lefthook_1.12.3_Linux_x86_64.gz | \
    gzip -d > /usr/local/bin/lefthook && \
    chmod +x /usr/local/bin/lefthook

RUN curl --proto '=https' --tlsv1.2 -fsSL https://get.opentofu.org/install-opentofu.sh -o install-opentofu.sh && \
    chmod +x install-opentofu.sh && \
    ./install-opentofu.sh  --install-method standalone && \
    rm ./install-opentofu.sh

USER vscode
