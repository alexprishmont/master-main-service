#!/bin/bash
PIN='123456'
MODULE_PATH='/usr/lib/softhsm/libsofthsm2.so'
LABEL='MyToken'

echo "Listing keys..."
pkcs11-tool --module $MODULE_PATH --login --pin $PIN --list-objects

KEY_IDS=$(pkcs11-tool --module $MODULE_PATH --login --pin $PIN --list-objects | grep 'label:' | awk '{print $2}')

echo "Found IDs: $KEY_IDS"

for ID in $KEY_IDS
do
    echo "Attempting to delete private key with ID $ID"
    pkcs11-tool --module $MODULE_PATH --label "$LABEL" --login --pin $PIN --delete-object --type privkey --label "$ID"
    echo "Attempting to delete public key with ID $ID"
    pkcs11-tool --module $MODULE_PATH --label "$LABEL" --login --pin $PIN --delete-object --type pubkey --label "$ID"
done
