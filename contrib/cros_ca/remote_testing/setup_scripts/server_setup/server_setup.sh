#!/bin/bash

# Step 1: Create .ssh directory if it doesn't exist
if [ ! -d "$HOME/.ssh" ]; then
    mkdir "$HOME/.ssh"
    echo "Created .ssh directory"
fi

# Step 2: Copy SSH keys to .ssh directory
cp ~/chromiumos/chromite/ssh_keys/testing_rsa ~/chromiumos/chromite/ssh_keys/testing_rsa.pub "$HOME/.ssh/"
echo "Copied testing_rsa and testing_rsa.pub to ~/.ssh"

# Step 3: Set permissions for testing_rsa
chmod 600 "$HOME/.ssh/testing_rsa"
echo "Changed permissions for testing_rsa"

# Step 4: Add testing_rsa to SSH agent
ssh-add "$HOME/.ssh/testing_rsa"
echo "Added testing_rsa to SSH agent"
