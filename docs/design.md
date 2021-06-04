# Design

This documentation aims to describe the controller logic. It can help to understand how does it work.

## Components

The chaos-controller is made of two main components:

* the controller handling the disruption resource lifecycle
* the injector, a CLI handling the disruption injection and cleanup into targets

## Disruption resource lifecycle

The lifecycle is divided in two phases:

* the injection phase happening on resource creation
* the cleanup phase happening on resource deletion

In between, there is a "waiting" phase where nothing happens.

### Injection phase (on disruption resource creation)

The injection phase takes care to create chaos pods to inject the described disruption.

#### Step 1: add a disruption finalizer

Adding a finalizer to the disruption resource will prevent it to be garbage collected on deletion, allowing the controller to properly take cleanup actions. It'll be removed by the cleanup phase later once we are sure that the disruption is fully cleared.

#### Step 2: compute spec hash

A hash of the disruption resource spec is computed and stored in the resource status. It is used later to detect any changes in the disruption spec. The resource being immutable (any changes made to it will have no effect), it allows to warn the user about that by recording an event in the resource.

#### Step 3: select targets

A list of targets is created from the given selector and level. It first lists the targets (pods if the given level is `pod`, nodes if it is `node`) matching the given label selector. Then, it randomly selects a certain amount of targets in the list depending on the given count. If the count is a percentage, it rounds up the amount.

The list of targets is then added to the disruption resource status. An event is recorded in each target to ease tracing/debugging.

#### Step 4: create chaos pods

For each target and disruption kind (network, disk pressure, cpu pressure, etc.), one chaos pod is created (running the injector image). A chaos pod is always scheduled on the same node as the target, but [will not be in the same namespace as the target](faq.md#Where-can-I-find-the-chaos-pods-for-my-disruption?). It will inject the disruption depending on the given parameters and will sleep, catching any exit signal (`SIGINT` or `SIGTERM`). A finalizer is also added to each chaos pod, preventing it to be garbage collected by Kubernetes during the cleanup phase.

#### Step 5: update injection status

The disruption injection status can take 3 different values:

* `NotInjected` when none of the chaos pods have successfully injected the disruption yet
* `PartiallyInjected` when at least one of the chaos pods has successfully injected the disruption
* `Injected` when all chaos pods have successfully injected the disruption

This status is being updated regularly until it reaches the `Injected` status. To evaluate if an injection went well or not, each chaos pod has a readiness probe looking for a file named `/tmp/readiness_probe`. This file is created by the injector when the injection is successful.

### Cleanup phase (on disruption resource deletion)

#### Step 1: clean disruptions

The controller deletes every chaos pod (not deleted yet) owned by the related disruption. Such a delete will trigger the reconcile loop again for this instance in order to handle the chaos pods' termination.

#### Step 2: handle chaos pods termination

**NOTE: this step is done at each reconcile loop call, not only on disruption deletion, so any chaos pods being deleted, either by the controller or by an external reason (like a node being evicted), will be handled.**

For each target, the controller checks if the target is still cleanable. A target is considered as not cleanable if it does not exist anymore or if it is not running. A non-cleanable target chaos pod will still be deleted, triggering the cleanup phase. However, its status won't be checked.

Then, each chaos pod of a given target will be treated like this (it can take multiple loops to reconcile correctly):
* if it is **completed** (exited successfully), **pending** (no injection happened or has been evicted) or **non-cleanable**, the finalizer of the chaos pod is **removed** allowing it to be garbage collected as soon as possible
* if it is **failed** (exited with an error) and **cleanable**, the chaos pod is kept for further investigation (and eventually manual cleaning) and the disruption is marked as stuck on removal (it won't be removed until manual actions are taken)

The disruption is considered as cleaned when there is no chaos pods left. For each reconcile call where the disruption is not fully cleaned, the reconcile request is re-enqueued.

## Injector lifecycle

The chaos pod uses the injector component. It is a CLI initializing a specific injector used to inject and clean a disruption. It has one subcommand per disruption kind (network disruption, cpu pressure, disk pressure, etc.). Whatever the used injector is, the lifecycle is always the same.

### Step 1: initialization

The injector initializes all the stuff common to all disruptions:

* the logger used to log from the injectors
* the metrics sink used to report metrics
* the injector configuration
  * it loads the targeted container (if injecting at the pod level)
  * it creates the cgroups manager (used by injectors to interact with the target cgroups)
  * it creates the network namespace manager (used by injectors to interact with the target network namespace)
* the exit signal handler used to catch `SIGINT` and `SIGTERM`

### Step 2: pre-run

It then enters the pre-run phase, initializing the injector itself depending on the given flags and with the previously initialized configuration. This is the only phase which is different for each injector.

### Step 3: run (inject and wait)

Once the injector is initialized, the injection starts. Once done, the injector creates the `/tmp/readiness_probe` file to validate the readiness probe and then sleeps listening to any signal arriving into the signal handler. At this point, nothing else will happen until a signal arrives, triggering the cleanup phase.

If an error occurs during the injection, it logs it but does not exit. It allows to injector to clean partially injected disruptions.

### Step 4: post-run (clean and exit)

When a signal arrives into the signal handler channel, it triggers the post-run phase which calls the injector clean method. Any error happening during the cleanup phase will make the injector to retry up to 3 times and, if the error is still occurring, to exit with a non-zero code, considering the chaos pod as "failed".
