FROM mcr.microsoft.com/devcontainers/go@sha256:379df9a272ed38aa7a735ad1980bdf6d9462517e40e821a21006e33b105079f7

RUN rm -f /etc/apt/sources.list.d/yarn.list /etc/apt/trusted.gpg.d/yarn.gpg
RUN curl -ssL "https://dl.k8s.io/release/v1.32.0/bin/linux/amd64/kubectl" -o kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

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

RUN curl -fsSL https://claude.ai/install.sh -o install-claude.sh && \
    chmod +x install-claude.sh && \
    ./install-claude.sh && \
    rm install-claude.sh
