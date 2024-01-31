#!/bin/bash

# Step 1: Create .ssh directory if it doesn't exist
if [ ! -d ".ssh" ]; then
    mkdir .ssh
    echo "Created .ssh directory"
fi

# Step 2: Move and rename testing_rsa.pub to ~/.ssh/authorized_keys
mv -f testing_rsa.pub ~/.ssh/authorized_keys
echo "Moved and renamed testing_rsa.pub to ~/.ssh/authorized_keys"
