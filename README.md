# Infrastructure for AI Trading Agents That Can't Afford Downtime

## The Problem I Identified

Senpi AI trading agents managing live positions face critical infrastructure failures on platforms like Railway:

1. **Pod evictions liquidate open positions** — Infrastructure events trigger SIGTERM during active trades, leaving capital unprotected
2. **Users must share AI API keys with the platform** — OpenAI/Anthropic keys stored in Railway environment variables = platform has full access. Regulatory red flag for fintech.
3. **SOC 2 requires Pro tier minimum** — Hobby ($5/month) has zero compliance coverage. [Pro ($20/month) has SOC 2](https://railway.com/pricing#:~:text=18%20months-,SOC%202%20compliance,-Granular%20access%20control), but doesn't solve the core problem: user secrets living on shared infrastructure without true isolation.
4. **No real multi-tenancy** — One agent's failure cascades; no namespace isolation or resource guarantees

When downtime means liquidations and compliance means custody of user credentials, infrastructure isn't optional—it's the product.

---

## Architecture Solution

```mermaid
graph TB
    subgraph "User Layer"
        User[User/App Team]
    end

    subgraph "Control Plane"
        User -->|kubectl apply| CRD[SenpiAgent CRD]
        CRD -->|Crossplane| Composition[Composition Engine]
    end

    subgraph "Secrets Pipeline - Zero Knowledge"
        Client[Client-Side Encryption] -->|Encrypted Keys| ASM[AWS Secrets Manager]
        ASM -->|External Secrets Operator| ESO[ESO]
        ESO -->|Inject to Pod| Secret[K8s Secret]
    end

    subgraph "EKS Cluster - Per-User Namespace"
        Composition -->|Provisions| NS[Namespace: user-alice]
        Composition -->|Provisions| SA[ServiceAccount]
        Composition -->|Provisions| SS[StatefulSet]
        
        Secret -.->|Mounted| SS
        
        SS --> Pod[Agent Pod]
        
        subgraph "Pod: Trading Agent"
            Container[OpenClaw Agent<br/>• Cron Jobs<br/>• Trading Logic<br/>• State Management]
            Terminator[Terminator Sidecar<br/>• PreStop Hook<br/>• Position Check<br/>• Block SIGTERM]
            PV[Persistent Volume<br/>Agent State<br/>10GB]
        end
        
        Pod --> Container
        Pod --> Terminator
        Pod -.->|Mounts| PV
    end

    subgraph "External Services"
        Container -->|Query Positions| HL[Hyperliquid API]
        Container -->|Decrypt & Use| AI[OpenAI/Anthropic]
        Terminator -->|Check Active Trades| HL
        Container -->|Audit Trail| CloudWatch[CloudWatch Logs]
    end

    subgraph "Observability"
        Pod -->|Metrics| Prometheus[Prometheus]
        Pod -->|Logs| Loki[Loki]
        Prometheus --> Grafana[Grafana Dashboards]
        Loki --> Grafana
    end

    style CRD fill:#9f7aea,color:#000
    style Composition fill:#9f7aea,color:#000
    style ASM fill:#f6ad55,color:#000
    style ESO fill:#f6ad55,color:#000
    style Secret fill:#f6ad55,color:#000
    style Terminator fill:#48bb78,color:#000
    style Container fill:#4299e1,color:#000
    style PV fill:#ed8936,color:#000
```

### Key Architecture Components

### 1. Crossplane Composition Layer
One API call creates everything. You define a `SenpiAgent` resource, Crossplane handles the rest:

```yaml
apiVersion: platform.senpi.ai/v1alpha1
kind:
metadata:
  name: user-88-agent
  namespace: default
spec:
  parameters:
    userId: "user-88"
    walletAddress: "0xc97ff0A66bC84FB8BcCEa34065af48d86be72B45"
    modelProvider: "openai"
    modelApiKey: "Xktb3BlbmFpLWtleQo="
    modelApiKeySalt: "dGVzdHNhbHQxMjM0NTY3OA=="
    modelName: "gpt-4"
    strategy: "striker"
```

**What Crossplane provisions automatically:**
- Dedicated namespace (`user-88`) > All agents gets deployed into it 
- ServiceAccount with RBAC
- Encrypted secrets from AWS Secrets Manager
- StatefulSet with persistent volume (10GB)
- Terminator sidecar for capital protection

**Real reconciliation in action:** Change the spec, Crossplane updates infrastructure. Delete the resource, everything gets cleaned up. No manual kubectl commands, no leftover resources.

This POC shows 4 core resources. Production would add: NetworkPolicies, PodDisruptionBudgets, HorizontalPodAutoscaler, monitoring ServiceMonitors, backup CronJobs, and more—all from one CRD.

**Proof it works:**

![Crossplane reconciliation creating SenpiAgent resource](assets/image-f6a45bfd-38d9-4f55-bb17-7576b1bfaab1.png)
*Single SenpiAgent resource triggers full stack provisioning*

![Composition creates all child resources automatically](assets/image-cb40763e-6183-4040-9780-c2cb2717c329.png)
*Crossplane composition provisions namespace, secrets, StatefulSet in sync*

![Full resource tree - namespace, ServiceAccount, secrets, StatefulSet](assets/image-8daa0a40-9078-4cae-a84a-181026326ab0.png)
*All child resources: namespace, ServiceAccount, secrets, StatefulSet running*

![Openclaw agent running ](assets/image.png)
*Openclaw agent running*
![Openclaw agent logs - real openclaw deployed but need to configure and modeify things for prodction ](assets/oc-running.png)
*Openclaw agent logs - real openclaw deployed but need to configure and modify i mean dummy creds addded for now*

**Deployment complexity: gone. App teams never touch Kubernetes.**

### 2. Terminator Sidecar — Capital Protection
PreStop hook that blocks pod termination during active Hyperliquid positions. 5-minute timeout prevents node deadlock. Infrastructure updates don't kill trades.

```go
func checkActivePositions() bool {
    // Query Hyperliquid API or local state
    // Block SIGTERM if positions are open
}
```

### 3. Zero-Knowledge Secrets Pipeline
**The Railway Problem:** Users paste OpenAI API keys into environment variables. Platform has full access. SOC 2 compliance available on Pro ($20/month), but Hobby users have no coverage. Even with SOC 2, Railway still holds plaintext secrets.

**The Solution:** Client-side encryption → AWS Secrets Manager → External Secrets Operator → pod injection. Keys never touch platform servers in plaintext. Audit trail included. SOC 2 compliance built into architecture regardless of tier.

---

## What This Enables

| Capability | Railway Hobby | Railway Pro/Enterprise | This Architecture |
|------------|---------------|------------------------|-------------------|
| **Pod shutdown safety** | ❌ Positions exposed | ❌ Positions exposed | ✅ Protected |
| **User API keys** | ❌ Platform has access | ❌ Platform has access | ✅ Zero-knowledge |
| **SOC 2 compliance** | ❌ Not available | ✅ Available | ✅ Built-in |
| **Real multi-tenancy** | ❌ Process isolation | ❌ Process isolation | ✅ Namespace isolation |

---

## Technical Stack

**Infrastructure:** EKS, Crossplane, Karpenter, VPC/IAM  
**Secrets:** AWS Secrets Manager, External Secrets Operator  
**Deployments:** GitHub Actions, ArgoCD, Helm, zero-downtime rollouts  
**Observability:** Prometheus, Grafana, Loki — SLOs for financial systems  
**Agent Runtime:** StatefulSets, persistent volumes, OpenClaw integration  
**Blockchain:** Hyperliquid API integration, wallet operations, position monitoring  

---

## Implementation Roadmap

1. Production EKS cluster + Crossplane control plane
2. Secrets pipeline: AWS Secrets Manager → ESO → pod injection
3. Terminator sidecar integration across agent fleet
4. Observability stack: alerting on position orphaning, state corruption, MCP auth expiry
5. Scale testing: dozens → thousands of concurrent agents
6. SOC 2 compliance audit preparation

---

**This infrastructure becomes the competitive moat.** Competitors lose traders because their infrastructure liquidates positions. This one doesn't.
