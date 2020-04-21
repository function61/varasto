#!/bin/bash -eu

# Imports seed data to be used for testing

cp misc/seed-data/varasto.db /tmp/varasto.db

# and client config (needed for server subsystems as well)
cp misc/seed-data/varastoclient-config.json /root/

# and also with sample data
tar -C /mnt -xf "misc/seed-data/blob-volumes.tar"
