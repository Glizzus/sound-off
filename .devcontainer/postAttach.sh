#!/bin/bash

cd .devcontainer || exit 1

if [ -f .env ]; then
    export "$(grep -v '^#' .env | xargs)"
fi

# If the user specifies a DigitalOcean access token, they get
# preconfigured with doctl.
if [ ! -z "$DIGITALOCEAN_ACCESS_TOKEN" ]; then
    doctl auth init

    # If the user has created a Kubernetes cluster using Terraform,
    # we automatically configure doctl to use it.
    # This assumes the cluster is named "soundoff-cluster".
    if doctl kubernetes cluster list -o json |
       jq -r '.[] | select(.name == "soundoff-cluster")' > /dev/null; then
        doctl kubernetes cluster kubeconfig save soundoff-cluster
    fi
fi

lefthook install
