# Senpi Crossplane Setup - Summary

## What We Built

Successfully set up a Crossplane-based infrastructure provisioning system for Senpi AI agents that automatically creates isolated Kubernetes resources per user.

## Architecture

### Crossplane Components

1. **XRD (CompositeResourceDefinition)**: `xsenpiagents.platform.senpi.ai`
   - Defines the API schema for SenpiAgent resources
   - Required fields: userId, walletAddress, modelProvider, modelApiKey, modelApiKeySalt
   - Optional fields: modelName (default: gpt-4), strategy (default: stalker)

2. **Composition**: `xsenpiagents.platform.senpi.ai`
   - Uses Pipeline mode with function-patch-and-transform
   - Creates 4 managed resources per claim:
     - Namespace (named after userId)
     - ServiceAccount (agent)
     - Secret (ai-model-credentials)
     - StatefulSet (agent pod with persistent storage)

3. **Claim**: `SenpiAgent`
   - User-facing API for requesting an agent
   - Example: `user-88-agent`

## Resources Created

When you apply a SenpiAgent claim, Crossplane automatically provisions:

```
user-88/                           (Namespace)
├── agent                          (ServiceAccount)
├── ai-model-credentials           (Secret with encrypted API keys)
├── user-88                        (StatefulSet)
│   └── user-88-0                  (Pod running akash202k/senpi:v1)
│       ├── MODEL_PROVIDER env
│       ├── MODEL_NAME env
│       ├── ENCRYPTED_API_KEY env
│       ├── API_KEY_SALT env
│       └── agent-state PVC (10Gi)
```

## Files Created

```
crossplane/
├── definition.yaml                 # XRD defining the API
├── composition.yaml                # How to provision resources
├── function.yaml                   # Patch-and-transform function
├── provider-kubernetes.yaml        # Kubernetes provider
├── provider-config.yaml            # Provider configuration
└── provider-kubernetes-rbac.yaml   # RBAC permissions for provider

k8s-manifests/
└── agent-template.yaml             # Example claim
```

## Installation Order

```bash
# 1. Install Crossplane (if not already installed)
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace

# 2. Install Kubernetes provider
kubectl apply -f crossplane/provider-kubernetes.yaml
kubectl wait --for=condition=Healthy provider/provider-kubernetes --timeout=300s

# 3. Configure provider RBAC
kubectl apply -f crossplane/provider-kubernetes-rbac.yaml

# 4. Configure provider
kubectl apply -f crossplane/provider-config.yaml

# 5. Install composition function
kubectl apply -f crossplane/function.yaml
kubectl wait --for=condition=Healthy function/function-patch-and-transform --timeout=300s

# 6. Apply XRD and Composition
kubectl apply -f crossplane/definition.yaml
kubectl apply -f crossplane/composition.yaml

# 7. Create agent claims
kubectl apply -f k8s-manifests/agent-template.yaml
```

## Verification

```bash
# Check claim status
kubectl get senpiagent

# Check composite resource
kubectl get xsenpiagent

# Check managed resources
kubectl get object -A | grep user-88

# Check actual Kubernetes resources
kubectl get all -n user-88

# Check pod logs
kubectl logs -n user-88 user-88-0
```

## Test Results

Successfully deployed `user-88-agent`:
- ✅ Namespace `user-88` created
- ✅ ServiceAccount `agent` created
- ✅ Secret `ai-model-credentials` with 4 keys created
- ✅ StatefulSet `user-88` running (1/1 replicas)
- ✅ Pod `user-88-0` running with persistent volume

## Security Features

1. **Encrypted API Keys**: User API keys stored encrypted with salt
2. **Namespace Isolation**: Each agent runs in its own namespace
3. **Secret Management**: Environment variables loaded from Kubernetes secrets
4. **Persistent Storage**: 10Gi PVC for agent state

## Next Steps

1. Implement encryption/decryption logic for API keys in your backend
2. Create a control plane API to generate SenpiAgent claims
3. Add monitoring and logging for agent pods
4. Configure resource limits and requests in the composition
5. Add network policies for additional security
6. Set up External Secrets Operator for vault integration (external-secrets.yaml exists)
