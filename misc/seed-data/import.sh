#!/bin/bash -eu

# Imports seed data to be used for testing

cp misc/seed-data/varasto.db /tmp/varasto.db

# and client config (needed for server subsystems as well)
mkdir -p /root/.config/varasto/
cp misc/seed-data/client-config.json /root/.config/varasto/

# and also with sample data
tar -C /mnt -xf "misc/seed-data/blob-volumes.tar"
