#!/bin/bash
set -e

ENVIRONMENT=${1:-staging}
echo "Deploying LinkFlow to $ENVIRONMENT..."

kubectl apply -k deployments/kubernetes/overlays/$ENVIRONMENT

echo "Waiting for deployments..."
kubectl rollout status deployment/gateway-service -n linkflow --timeout=300s

echo "Deployment complete!"
kubectl get pods -n linkflow
