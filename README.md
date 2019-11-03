# cluster-api-provider-plunder

_Pronounced_: Clust**ARRRR**-**APIARRRR**-provider-plunder.

**Warning**: This provider is so untested that I would only recommend using it against your worst enemies, that being said.. if you're feeling brave then be my guest :D

## What is it?

The `cluster-api-provider-plunder` is a [Cluster-API](https://github.com/kubernetes-sigs/cluster-api) provider that extends the capabilities of a Kubernetes cluster so that it can be used to not only manage the provisioning of applications and services, but also the provisioning of actual infrastructure to host additional Kubernetes clusters. 

This provider does this by "translating" infrastructure requests from Cluster-API and using [plunder](https://github.com/plunder-app/plunder) to provision `clusters` and the required `machines` that make up a complete Kubernetes cluster.

## How it works

Cloud environments typically have a **massive** agile advantage when it comes to provisioning infrastructure, typically the ground work (bare-metal provisioning) it already taken care of (I'm presuming through the automation of `api-human`). Meaning that when you ask for some infrastructure then the cloud provider will be cloning VM templates etc.. to quickly get some machine infrastructure back to you. 

### Bare-Metal

In order to handle quick provisioning today `plunder` watches for machines starting up that typically are blank/new and need to try and boot, it will register their MAC addresses and place them in a reboot loop (until we need them).

```
$ pldrctl get unleased
Mac Address        Hardware Vendor  Time Seen                 Time since
00:50:56:a5:11:20  VMware, Inc.     Sun Nov  3 10:54:18 2019  0s
00:50:56:a5:b5:f1  VMware, Inc.     Sun Nov  3 10:54:18 2019  0s
```

**NOTE:** There are more efficent ways of doing this, just haven't had the time `¯\_(ツ)_/¯`

When it comes to provisioning, we simply flip a server from `reboot` to `provision` and "hey presto"

## Using it

At the moment, there is still a few steps that are needed to get this all up and running and i've yet to get the provider in a kubernetes deployment (someone feel free to raise an issue). 

### Install CRDs

`make install`

Then verify them with `kubectl get crds | grep plunder`. 

### Install/Run Controller

Copy the `plunderclient.yaml` file to the same location that the controller will run.

`make run` will then start the controller.

## Deploy Manifests

### Cluster Definition

Cluster.yaml should typically look like below (the `cidrBlocks` are unimplemented from my side currently).

```
apiVersion: cluster.x-k8s.io/v1alpha2
kind: Cluster
metadata:
  name: cluster-plunder
spec:
  clusterNetwork:
    pods:
      cidrBlocks: ["192.168.0.0/16"]
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: PlunderCluster
    name: cluster-plunder
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: PlunderCluster
metadata:
  name: cluster-plunder
```

### Machine Definition

**IPAM** isn't completed (lol.. it's not started), so currently you'll need to specify addresses for machines. This will need fixing for `machineSets`

Machine.yaml should looks something like below:

```
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: PlunderMachine
metadata:
  name: controlplane
  namespace: default
spec:
  ipaddress: 192.168.1.123
---
apiVersion: cluster.x-k8s.io/v1alpha2
kind: Machine
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: cluster-plunder
    cluster.x-k8s.io/control-plane: "true"
  name: controlplane
  namespace: default
spec:
  version: "v1.14.2"
  bootstrap:
    data: ""
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: PlunderMachine
    name: controlplane
    namespace: default
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: PlunderMachine
metadata:
  name: worker
  namespace: default
spec:
  ipaddress: 192.168.1.124
---
apiVersion: cluster.x-k8s.io/v1alpha2
kind: Machine
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: cluster-plunder
  name: worker
  namespace: default
spec:
  version: "v1.14.2"
  bootstrap:
    data: ""
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: PlunderMachine
    name: worker
    namespace: default
```

## Deploy in Kubernetes

The same manifests are in `examples/simple` and can be deployed through `kubectl` with the command:

`kubectl -f examples/simple/cluster.yaml` and `kubectl create -f examples/simple/machine.yaml`

### Watching the deployment

#### Machine State

```
k get machines
NAME           PROVIDERID   PHASE
controlplane                provisioning
worker                      provisioning
```

#### Machine Events

```
k get events
LAST SEEN   TYPE      REASON              OBJECT                        MESSAGE
41m         Warning   No Hardware found   plundermachine/controlplane   Plunder has no available hardware to provision
41m         Normal    PlunderProvision    plundermachine/controlplane   Plunder has begun provisioning the Operating System
34m         Normal    PlunderProvision    plundermachine/controlplane   Host has been succesfully provisioned OS in 6m56s Seconds
9m40s       Warning   No Hardware found   plundermachine/controlplane   Plunder has no available hardware to provision
7m4s        Normal    PlunderProvision    plundermachine/controlplane   Plunder has begun provisioning the Operating System
3s          Normal    PlunderProvision    plundermachine/controlplane   Host has been succesfully provisioned OS in 7m1s Seconds
31m         Warning   No Hardware found   plundermachine/worker         Plunder has no available hardware to provision
16m         Normal    PlunderProvision    plundermachine/worker         Plunder has begun provisioning the Operating System
9m58s       Normal    PlunderProvision    plundermachine/worker         Host has been succesfully provisioned OS in 7m1s Seconds
2s          Normal    PlunderProvision    plundermachine/worker         Plunder has begun provisioning the Operating System
```

## Deleting Machines

There are two methods for removing the deployed machines:

`kubectl delete machines --all` or `kubectl delete -f ./examples/simple/machine.yaml`

This process will wipe the boot sector and beginning of the disk which will result in it booting into a "blank enough" state for plunder to add it back to the reboot loop.
