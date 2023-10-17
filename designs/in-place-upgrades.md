# In-place upgrades
## Introduction
At present, the only supported upgrade strategy in EKS-A is rolling update. However, for certain use cases (such as Single-Node Clusters with no spare capacity, Multi-Node Clusters with VM/OS customizations, etc.), upgrading a cluster via a Rolling Update strategy could either be not feasible or a costly operation (requiring to add new hardware, re-apply customizations...).

In-place upgrades aims at solving this problem by allowing users to perform Kubernetes node upgrades without replacing the underlying machines.

### About this doc
This is a a high level design doc. It doesn't answer all the questions or give a complete implementation spec. Instead, it presents a high level architecture: the communication patterns and dependency between all components and an a concrete implementation for each new component.

The goal is to bring enough clarity to the table that a general solution can be agreed upon so the work can be split by component. Each of these components might present their own challenges that will need to be addressed, at their own time, by their respective owners (with or without a design doc).

## Goals and Objectives
### User stories
- As a cluster operator, I want to upgrade (EKS-A version and/or K8s version) my baremetal clusters without loosing any OS changes I have made to the machines and  without needing extra unused hardware.
- As a cluster operator, I want to in-place upgrade (EKS-A version and/or K8s version) my baremetal clusters in air-gapped environments.
- As a cluster operator, I want to in-place upgrade (EKS-A version and/or K8s version) my single node baremetal clusters.

### Out of scope
* External etcd

## Overview of Solution
This section will walk from high level idea to concrete implementation. It'll start with a rough component and interaction overview, then list the remaining problems/questions this solution leaves to be solved and then finish with the individual solutions for each of these questions.

### CAPI implementation goals
- Avoid "hardcoding" all the new upgrade logic in the existing core/kubeadm controllers.
	- Different users might have different preferences on how to perform these upgrade operations and it seems unreasonable to expect capi to meet them all. Requirements around security, access, audit, tooling, OS, infrastructure etc. can variate substantially from environment to environment.
	- Moreover, certain users might already have "provisioning" systems they might want to leverage.
- Maintain a coherent UI/UX between rollout upgrades and in place upgrades. Eg. an in-place upgrade must be triggered in the same way as an immutable upgrade.
- Users should be able to keep their cluster running basically indefinitely just with in-place upgrades.

### Component diagram
**TLDR**: CAPI still decides when an upgrade is required but it delegates to an external component the work to perform it. After the upgrade process is completed, `Machine`s (and maybe `BootstrapConfig`) objects are updated to reflect the k8s component level changes.

![System](images/In-place-upgrades.excalidraw.png)

Both KCP and MachineDeployments have a configurable strategy to be followed when updates are required: it first follows a common mechanism to detect when an upgrade is required and then it reads the configured strategy to follow one process or another.

Given the infrastructure's immutability constraints, this is currently scoped to just machine replacement. Simplifying here, these existing strategies allow to control if new machines are created before or after the old ones are deleted. What if in-place upgrades can be just another strategy? That fits the current API and abstractions quite well. Moreover, what if the logic to follow the strategy wasn't implemented by these two controllers at all?

This solution decouples the core CAPI controllers from the individual possible strategies that can be followed for an in place upgrade. CAPI users could choose to implement their own strategy: this includes us. In addition, we could iterate on them without requiring changes in CAPI. We would even allow multiple strategies available in the same CAPI installation for different types of nodes/clusters. 

From the EKS-A side, the only thing we would need to configure in the KCP/Machine deployment is a `strategy: external`. This solution allows to leverage the existing logic in these two controllers to determine when changes are required, so this doesn't need to be replicated elsewhere. Once the need for changes has been determined, these controllers will hit the registered hooks informing of the selected Machines and the computed difference between current Machine and desired Machine. The hooks would take over the process and perform the necessary actions.

Note: the external external strategy pattern is not tied to just in-place upgrades: it can be used to change the machine upgrade order, fine control the timing of each machine upgrade, etc. On EKS-A we will just implement an strategy that upgrades machines in place.

The following diagram presents the same idea as above but with a concrete example for the KCP and in a different format, just as additional clarification.

```mermaid
sequenceDiagram
autonumber
participant User

box Management Cluster
participant apiserver as kube-api server
participant kcp as KCP controller
participant hook as External upgrader
end

box Workload Cluster
participant infra as Infrastructure
end


User->>+apiserver: change KCP's K8s version <br>(external strategy)
apiserver->>+kcp: Notify changes
apiserver->>-User: OK
kcp->>+hook: Upgrade required
hook->>kcp: OK
deactivate kcp
hook->>infra: Upgrade k8s componentes
alt opt 1
	hook->>apiserver: Mark machine as updated
else opt 2
	kcp->>apiserver: Mark machine as updated
end
```

#### Remaining questions
Following the first diagram, all points marked with a ❓:
1. How does CAPI request an upgrade from the external strategy implementer and what information needs to provide?
2. Where does EKS-A implement the in-place upgrade strategy?
3. How does the EKS-A in-place strategy upgrade the components in the running nodes?
4. Who is responsible for updating the CAPI `Machine` objects so the rest of controllers detect the upgrade has been completed?
	1. Option 1: the CAPI KCP/MachineDeployment controllers. The external strategy will need a way to signal this.
	2. Option 2: the external strategy implementer that accepted the upgrade request.

1 and 4 are decisions that need to be made at the CAPI level, since they conform the "external strategy contract". 2 and 3 are decoupled from CAPI and EKS-A can freely decide how to solve them and/or change them later (as long as they still allow to conform to the given contract).

**Warning**: the idea of using an external strategy and the below proposed solutions for 1 and 4 are subject to change since they still have to be accepted by the CAPI community.

### CAPI external strategy contract
#### Communication pattern
There two obvious options: webhooks or the CRDs + controller model. The tradeoffs are mostly the same as in any other system, so this doc won't enumerate them.

CAPI already already uses both:
- CRD/contract based extension for infra, CP and bootstrap providers. These are "plug-in, swappable low-level components" (quoting the Runtime SDK proposal)
- Webhooks for the [Runtime SDK](https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20220221-runtime-SDK.md). The Runtime Hooks were design to enable "systems, products and services built on top of Cluster API that require strict interactions with the lifecycle of Clusters, but at the same time they do not want to replace any low-level components in Cluster API, because they happily benefit from all the features available in the existing providers (built on top vs plug-in/swap)".

A external upgrade strategy seems to fit well in both categories: it's both a *plug-in, swappable low-level component* and it also *does not want to replace any low-level components in Cluster API, because it happily benefits from all the features available in the existing providers*.

We propose to use the Runtime Hooks and Extensions (hence webhooks) because: 
- They align quite well with our goals in two main areas (quoting the proposal):
	- "it will be also possible to use this capability for allowing the user to hook into Cluster API reconcile loops at "low level", e.g. by allowing a Runtime Extension providing external patches to be executed on every topology reconcile". This is exactly what we are trying to do here.
	- "The proposed solution is designed with the intent to make developing Runtime Extensions as simple as possible, because the success of this feature depends on its speed/rate of adoption in the ecosystem."
- Using webhooks doesn't prevent strategy implementers from using a controller.
- It facilitates enforcing an API contract.

#### External upgrade strategy Request
At a minimum, the request initiated by CAPI needs to include:
- List of machines that require an upgrade.
- New `Machine` spec.
- New Bootstrap config spec.
- New Infra Machine spec.
- A reference to the `Cluster`.
- A reference to the owner of the Machines that requests the upgrade (CP object or `MachineDeployment`).

#### External upgrade strategy Response
Based on the proposed pattern, there is no information that the implementers need to communicate back in the response to CAPI. We can't really make the operation synchronous since upgrades can take an undermined amount of time, so the only thing the implementer needs to do is communicate with a 200 OK if it has successfully "started" the process and the appropriate error otherwise.

However, we propose an extension of this pattern that although simple in its implementation, it eliminates the requirement for complexity in the external implementers. 

What if the changes detected by CAPI are not scoped down to just a k8s version upgrade? They could involve something else, like running certain extra commands in the host, additional mount points or even configuring users. For example, the current kubeadm config API mixes host level and k8s configuration, which means our in-place upgrade extensions would need to implement a mechanism to reconcile these types of changes as well. This not only can be quite complex but it also can greatly variate from OS to OS. Moreover, what if the detected changes are at the infrastructure Machine level? Depending on the provider, it might even be impossible to make these changes without replacing the machine.

To add some flexibility for the strategy extensions implementers, we will add the option for hooks to decline an upgrade request if they decide they are not capable of performing it. We will also add an option in the KCP/MachineDeployment to configure a `fallbackRolling`, so if all the hooks respond with a "won't do", the controller will continue the upgrade process following this fallback replace strategy. This pattern allows strategy extensions to decide how much of all the possible in-place upgrade scenarios they want to cover and let the system still be able to reconcile any other scenarios they don't cover.

```mermaid
sequenceDiagram
autonumber
participant User

box Management Cluster
participant apiserver as kube-api server
participant kcp as KCP controller
participant hook as In place upgrade extension
end

box Workload Cluster
participant infra as Infrastructure
end

User->>+apiserver: change KCP's K8s version <br>(external strategy and fallback) 
apiserver->>+kcp: Notify changes
apiserver->>-User: OK
kcp->>+hook: Upgrade request
hook->>-kcp: Reject
kcp->>infra: Create machine
kcp->>-infra: Delete old machine
```

The Response will include at a minimum:
- Boolean indicated of the request is accepted or not.
- A reason message in case the request is declined.

#### Updating CAPI objects after upgrade
We propose putting this responsibility on the strategy implementers for two reasons:
- It barely increases the complexity of the implementers: they already know when they process is completed and should be capable of identifying the fields and objects that need to be updated (since they need to decide id the requested change is supported or not).
- It simplifies the overall system avoiding a back channel from strategy implementers to CAPI. The CAPI components (KCP/Machine deployment controllers) will simply sit in a loop, detecting that a machine needs to be upgraded until the implementer updates the appropriate objects. Then the corresponding KCP/MachineDeployment will be marked as ready.

```mermaid
sequenceDiagram
autonumber
participant User

box Management Cluster
participant apiserver as kube-api server
participant kcp as KCP controller
participant hook as In place upgrade extension
end

box Workload Cluster
participant infra as Infrastructure
end

User->>+apiserver: change KCP K8s version <br>(external strategy)
apiserver->>+kcp: Notify changes
apiserver->>-User: OK
kcp->>+hook: Upgrade request
hook->>kcp: OK
deactivate kcp
hook->>infra: Update k8s components
hook->>apiserver: Mark machine as updated
activate kcp
kcp->>apiserver: Mark KCP as Ready
deactivate kcp
```

### EKS-A in-place upgrade strategy implementation
TODO: 
- use a controller and CRDs for the strategy.
	- 3 CRD/controllers: `ControlPlaneUpgrade`, `MachineDeploymentUpgrade` and `NodeUpgrade`.
	- Maybe controller for `NodeUpgrade`, maybe not. If yes, we need to pass the PodSpec so we can have different processes for CP and workers.
- Use priviledged pods with nsenter that are scheduled in the target nodes. Separate kubeadm steps in initContainers so the process can we followed easier.

TODO: maybe add C4 system diagram with mermaid.

## Customer experience
TODO: talk about default strategy and EKS-A API changes. Talk about how to debug failures: look to CRDs, pod logs, etc.

## Security
TODO: talk about privileged containers and the use of nsenter.

## Testing
TODO: nothing special here. This will need their own testing on the CAPI side, mostly a fake strategy to be tested. And the classic unit and e2e testing on the EKS-A side.

## Appendix
### Technical specifications
- Support only for Ubuntu.
- Support for single node clusters without extra hardware. This means cleaning up + reinstalling won’t work, since the etcd data will be contained only on this node.
- Downtime of workloads is acceptable. It’s up to the user to configure them in a way that the are resilient to having one node down.
- Given the requirements, in some cases (like single node), CP downtime is acceptable.
- HA control planes should not have API server downtime, except for however long kube-vip takes to do the fail over (for clusters using kube-vip). If the user is running custom workloads in the CP nodes, these might have downtime if they are not enough compute resources to run them when one node is down.
- Support for air-gap. This means that all the components to be upgraded in a node need to packaged and stored somewhere.
- `apt` updates and any other OS level customization by the customer are supported, but only on components not managed by eks-a.
- Host components owned by eks-a 
	- container runtime
	- CNI plugins
	- kubeadm
	- kubelet
	- kubectl
	- CP static pods (as in any other eks-a cluster)
- EKS-A will only manage upgrading the eks-a managed components. Everything else like OS, additional packages, etc. will be handled by the user.
- In the EKS-A API the default upgrade strategy should still be the current rolling upgrade.
- Nodes should follow the kubernetes setup of image builder for ubuntu: kubelet running with `systemctl`, `containerd` as the container runtime, etc.