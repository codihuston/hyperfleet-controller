#!/bin/bash

# HyperFleet Sample Deployment Script
# This script applies the HypervisorCluster sample with environment variable substitution

set -e

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Error: .env file not found!"
    echo "Please copy .env.example to .env and fill in your values:"
    echo "  cp .env.example .env"
    echo "  # Edit .env with your actual values"
    exit 1
fi

# Load environment variables
echo "Loading environment variables from .env..."
export $(grep -v '^#' .env | xargs)

# Set defaults for optional variables
export CLUSTER_NAME=${CLUSTER_NAME:-proxmox-test}
export NAMESPACE=${NAMESPACE:-default}
export NODE_1=${NODE_1:-pve-node-1}
export DEFAULT_STORAGE=${DEFAULT_STORAGE:-local-lvm}
export DEFAULT_NETWORK=${DEFAULT_NETWORK:-vmbr0}
export DNS_DOMAIN=${DNS_DOMAIN:-hyperfleet.local}
export DNS_SERVER_1=${DNS_SERVER_1:-192.168.1.1}
export DNS_SERVER_2=${DNS_SERVER_2:-8.8.8.8}
export SECRET_NAME=${SECRET_NAME:-test-proxmox-credentials}
export ENVIRONMENT=${ENVIRONMENT:-test}

# Validate required variables
required_vars=("PROXMOX_ENDPOINT")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "Error: Required environment variable $var is not set in .env file"
        exit 1
    fi
done

echo "Applying HypervisorCluster sample with the following configuration:"
echo "  Cluster Name: ${CLUSTER_NAME}"
echo "  Namespace: ${NAMESPACE}"
echo "  Proxmox Endpoint: ${PROXMOX_ENDPOINT}"
echo "  Nodes: ${NODE_1}"
echo ""

# Apply the template with environment variable substitution using envsubst
envsubst < config/samples/hypervisor_v1alpha1_hypervisorcluster_template.yaml | kubectl apply -f -

echo "HypervisorCluster sample applied successfully!"
echo ""
echo "Check the status with:"
echo "  kubectl get hypervisorclusters"
echo "  kubectl describe hypervisorcluster ${CLUSTER_NAME}"