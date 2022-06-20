# Chaos Controller Features Guide

## Dry-run mode

 This fakes the injection while still going through the process of selecting targets, creating chaos pods, and simulating the disruption as much as possible. Put a different way, all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets). Checkout this [example](../examples/dry_run.yaml).

## Level

A disruption can be applied either at the `pod` level or at the `node` level:

* When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted may still be affected depending on the injected disruption.
* When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other processes).

### Example

Let's imagine a node with two pods running: `foo` and `bar` and a disruption dropping all outgoing network packets:

* Applying this disruption at the `pod` level and with a selector targeting the `foo` pod will result with the `foo` pod not being able to send any packets, but the `bar` pod will still be able to send packets, as well as other processes on the node.
* Applying this disruption at the `node` level and with a selector targeting the node itself, both `foo` and `bar` pods won't be able to send network packets anymore, as well as all the other processes running on the node.

## Duration

The `Disruption` spec takes a `duration` field. This field represents amount of time after the disruption's creation before 
all chaos pods automatically terminate and the disruption stops injecting new ones. This field takes a string, which is meant to conform to 
golang's time.Duration's [string format, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration) This time is measured from the moment 
that the Disruption resource is created, not from when the injection of the actual failure occurs. It functions as a strict maximum for the lifetime of the Disruption, not a guarantee of how long the failure will persist for.

If a `duration` is not specified, then a disruption will receive the default duration, which is configured at the controller level by setting 
`controller.defaultDuration` in the controller's config map, and this value defaults to 1 hour.

After a disruption's duration expires, the disruption resource will live in k8s for a default of 10 minutes. This can be configured by altering 
`controller.expiredDisruptionGCDelay` in the controller's config map.

## Pulse

The `Disruption` spec takes a `pulse` field. It activates the pulsing mode of the disruptions of type `cpu_pressure`, `disk_pressure`, `dns_disruption`, `grpc_disruption` or `network_disruption`. A "pulsing" disruption is one that alternates between an active injected state, and an inactive dormant state. Previously, one would need to manage the Disruption lifecycle by continually re-creating and deleting a Disruption to achieve the same effect.

It is composed of two subfields: `dormantDuration` and `activeDuration`, which both take a string, which is meant to conform to 
golang's time.Duration's [string format, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration) and **have to be greater than 500 milliseconds**.

`dormantDuration` will specify the duration of the disruption being `dormant`, meaning that the disruption will not be injected during that time.

`activeDuration` will specify the duration of the disruption being `active`, meaning that the disruption will be injected during that time.

The pulsing disruption will be injected for a duration of `activeDuration`, then be clean and dormant for a duration of `dormantDuration`, and so on until the end of the disruption.

If a `pulse` is not specified, then a disruption will not be pulsing.

## Targeting

**NEW:** StaticTargeting currently defaults to true. It will default to false after some transition time. [Read StaticTargeting](#StaticTargeting-(current-default-behaviour)).

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. It's possible to specify multiple label selectors, in which case the controller will select from targets that match all of them. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

### StaticTargeting (current default behaviour)

When `StaticTargeting` is activated, targeting is limited to a single target selection at the disruption's creation. It allows for more controlled disruption impact and propagation, as the targets will never change and _can_ be compensated for in case they are made useless. Its major limit is not being able to follow targets through deployments/rollouts.
By setting the `StaticTargeting` flag to `false` in the [Disruption's yaml spec](../examples/static_targeting.yaml), you activate a constant re-targeting for this disruption. This means at any given time, any target within the selector's scope will be added to the target list and be disrupted.
This is a feature to use with care, as it can quickly get out of control: per example, a disruption targeting 100% of an application's pod will affect all existing **and** future pods which can appear once the disruption started. As long as this 100% disruption exists, there will be no spared pod.
Even a 50%-set disruption, if the disruption gets targeted pods to die, will constantly re-target 50% of all selector-fitting pods and end up killing every pod unless they are recreated.

`DynamicTargeting` behavior design choices:
- the controller will consider as a still-alive target any pod that exists - regardless of its state.
- the controller will reconcile/update its targets list on any chaos pod or selector movement (create, update, delete)

### Targeting safeguards

When enabled [in the configuration](../chart/values.yaml) (`controller.enableSafeguards` field), safeguards will exclude some targets from the selection to avoid unexpected issues:

* if the disruption is applied at the node level, the node where the controller is running on can't be selected
* if the disruption is applied at the pod level with a node disruption, the node where the controller is running on can't be selected

### Advanced targeting

In addition to the simple `selector` field matching an exact key/value label, one can do some more advanced targeting with the `advancedSelector` field. It uses the [label selector requirements mechanism](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#LabelSelectorRequirement) allowing to match labels with the following operator:

* `Exists`: the label with the specified key is present, no matter the value
* `DoesNotExist`: the label with the specified key is not present
* `In`: the label with the specified key has a value strictly equal to one of the given values
* `NotIn`: the label with the specified key has a value not matching any of the given values

You can look at [an example of the expected format](../examples/advanced_selector.yaml) to know how to use it.

### Targeting a specific pod

How can you target a specific pod by name, if it doesn't have a unique label selector you can use? The `Disruption` spec doesn't support field selectors at this time, so selecting by name isn't possible. However, you can use the `kubectl label pods` command, e.g., `kubectl label pods $podname unique-label-for-this-disruption=target-me` to dynamically add a unique label to the pod, which you can use as your label selector in the `Disruption` spec.

### Targeting a specific container within a pod

By default, a disruption affects all containers within the pod. You can restrict the scope of the disruption to a single container or to only some containers [like this](../examples/containers_targeting.yaml).

## Applying a disruption on pod initialization

> :memo: This mode has some restrictions:
>   * it requires a 1.15+ Kubernetes cluster
>   * it requires the `--handler-enabled` flag on the controller container
>   * it only works for network related (network and dns) disruptions
>   * it only works with the pod level
>   * it does not support containers scoping (applying a disruption to only some containers)

It can be handy to disrupt packets on pod initialization, meaning before containers are actually created and started, to test startup dependencies or init containers. You can do this in only two steps:

* redeploy your pod with the specific label `chaos.datadoghq.com/disrupt-on-init` to hold it in the initialization state
  * the chaos-controller will inject an init containers name `chaos-handler` as the first init container in your pod
  * this init container is lightweight and does nothing but waiting for a `SIGUSR1` signal to complete successfully
* apply your disruption [with the init mode on](../examples/on_init.yaml)
  * the chaos pod will inject the disruption and unstuck your pod from the pending state

Note that in this mode, only pending pods with a running `chaos-handler` init container and matching your labels + the special label specified above will be targeted. The `chaos-handler` init container will automatically exit and fail if no signal is received within the specified timeout (default is 1 minute).

## Notifier

When creating a disruption, you may wish to be alerted of important lifecycle warnings (disruption found no target, chaos pod is stuck on removal, target is failing, target is recovering, etc.) through the Notifier module of the chaos-controller. On each occurence, these events will be propagated through the different set up notifiers (currently `noop/console`, `slack` and `datadog` are implemented).

You can find the complete list of the events sent out by the controller [here](/api/v1beta1/events.go#L24).

Any setup/config error will be logged at controller startup.

### Slack

The `slack` notifier requires a slack API Token to connect to your org's slack workspace. It will use the disruption's creator username in kubernetes (based on your authentication method) as an email address to send a DM on slack as 'Disruption Status Bot'. **The email address used to authentify on the kubernetes cluster and create the disruption needs to be the same used on the slack workspace** or the notification will be ignored.

### Datadog

The `datadog` notifier requires the `STATSD_URL` environment variable to be set up. It will either send a `Warn` event for warning kubernetes events or a `Success` event for normal recovered kubernetes events sent out by the controller.

### HTTP

The `http` notifier requires a `URL` to send the POST request to and optionally ask for the filepath of a file containing the headers to add to the request if needed. It will send a json body containing the notification information.

The file containing the headers is of format:

```
key1:value1
key2:value2
```

You can deploy an nginx server which will receive the http requests at [examples/http-notifier-in-demo.yaml](../examples/http-notifier-in-demo.yaml).

### Configuration

Please setup the following fields to `chart/templates/configmap.yaml - data - config.yaml - controller` pre-controller installation: 

```yaml
notifiers:
  common:
  	clusterName: <cluster name> # will be n/a otherwise
  noop:
  	enabled: true/false # enables the noop notifier
  slack:
  	enabled: true/false # enables the slack notifier
  	tokenFilepath: <slack token file path> # path to a file containing an API token for your slack workspace
  datadog:
    enabled: true/false # enables the datadog notifier
  http:
    enabled: true/false # enables the http notifier
    url: <url>
    headersFilepath: <headers file path> # path to a file containing the list of headers to add to the http POST request we send for the http notifier
      
```

## Disruption Examples

Please take a look at the different disruptions documentation linked in the table of content for more information about what they can do and how to use them.

Here is [a full example of the disruption resource](../examples/complete.yaml) with comments. You can also have a look at the following use cases with examples of disruptions you can adapt and apply as you wish:

* [Node disruptions](/docs/node_disruption.md)
  * [I want to randomly kill one of my node](../examples/node_failure.yaml)
  * [I want to randomly kill one of my node and keep it down](../examples/node_failure_shutdown.yaml)
* [Pod disruptions](/docs/container_disruption.md)
  * [I want to terminate all the containers of one of my pods gracefully](../examples/container_failure_all_graceful.yaml)
  * [I want to terminate all the containers of one of my pods non-gracefully](../examples/container_failure_all_forced.yaml)
  * [I want to terminate a container of one of my pods gracefully](../examples/container_failure_graceful.yaml)
  * [I want to terminate a container of one of my pods non-gracefully](../examples/container_failure_forced.yaml)
* [Network disruptions](/docs/network_disruption.md)
  * [I want to drop packets going out from my pods](../examples/network_drop.yaml)
  * [I want to corrupt packets going out from my pods](../examples/network_corrupt.yaml)
  * [I want to add network latency to packets going out from my pods](../examples/network_delay.yaml)
  * [I want to restrict the outgoing bandwidth of my pods](../examples/network_bandwidth_limitation.yaml)
  * [I want to disrupt packets going to a specific host, port or Kubernetes service](../examples/network_filters.yaml)
* [CPU pressure](/docs/cpu_pressure.md)
  * [I want to put CPU pressure against my pods](../examples/cpu_pressure.yaml)
* [Disk pressure](/docs/disk_pressure.md)
  * [I want to throttle my pods disk reads](../examples/disk_pressure_read.yaml)
  * [I want to throttle my pods disk writes](../examples/disk_pressure_write.yaml)
* [DNS resolution mocking](/docs/dns_disruption.md)
  * [I want to fake my pods DNS resolutions](../examples/dns.yaml)
* Network and DNS disruptions
  * [I want to disrupt network packets on pod initialization](../examples/on_init.yaml)
