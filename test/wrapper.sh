#!/bin/bash
set -e

echo "Starting UPF with configuration from upf.yaml"
cat /config/upf.yaml

# In a real implementation, this would parse the upf.yaml file
# and configure the UPF accordingly

# For demonstration purposes, we'll just sleep to keep the container running
echo "UPF is running..."
while true; do
  sleep 30
done 