#!/bin/bash

sudo chown -R vscode:vscode /go/pkg/mod

mkdir -p ~/.aws
cp -f .devcontainer/aws_credentials ~/.aws/credentials
chmod 600 ~/.aws/credentials

key_base="soundoff-infra"
priv=".devcontainer/$key_base"
pub=".devcontainer/$key_base.pub"

gen_pub=$(ssh-keygen -y -f "$priv" 2>/dev/null | awk '{print $1, $2}')
act_pub=$(awk '{print $1, $2}' "$pub")

if [[ "$gen_pub" != "$act_pub" ]]; then
  echo "Public key does not match private key."
  exit 1
fi

mkdir -p ~/.ssh
cp -f "$priv" "$pub" ~/.ssh/
chmod 600 ~/.ssh/$key_base

# We put .claude.json into .claude/ so that we can keep it in our cached volume.
# We just symlink ~/.claude.json (where Claude looks for it)
if [ ! -f /home/vscode/.claude/.claude.json ]; then
    echo "{}" > /home/vscode/.claude/.claude.json
fi

ln -sf /home/vscode/.claude/.claude.json /home/vscode/.claude.json