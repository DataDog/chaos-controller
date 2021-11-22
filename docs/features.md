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
golang's time.Duration's [string format, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration)

If a `duration` is not specified, then a disruption will receive the default duration, which is configured at the controller level by setting 
`controller.defaultDuration` in the controller's config map, and this value defaults to 1 hour.

After a disruption's duration expires, the disruption resource will live in k8s for a default of 15 minutes. This can be configured by altering 
`controller.expiredDisruptionGCDelay` in the controller's config map.

## Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. It's possible to specify multiple label selectors, in which case the controller will select from targets that match all of them. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

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

When creating a disruption, you may wish to be alerted of important lifecycle warnings (disruption found no target, chaos pod is stuck on removal, etc.) through the Notifier module of the chaos-controller. On each occurence, these events will be propagated through the different set up notifiers (currently `noop/console` and `slack` are implemented).

Any setup/config error will be logged at controller startup.

### Slack

The `slack` notifier requires a slack API Token to connect to your org's slack workspace. It will use the disruption's creator username in kubernetes (based on your authentication method) as an email address to send a DM on slack as 'Disruption Status Bot'. **The email address used to authentify on the kubernetes cluster and create the disruption needs to be the same used on the slack workspace** or the notification will be ignored.

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
