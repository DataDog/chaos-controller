# Chaos Controller Features Guide

## Dry-run mode

This fakes the injection while still going through the process of selecting targets, creating chaos pods, and simulating the disruption as much as possible. Put a different way, all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets). Checkout this [example](../examples/dry_run.yaml).

## Level

A disruption can be applied either at the `pod` level or at the `node` level:

- When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted may still be affected depending on the injected disruption.
- When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other processes).

### Example

Let's imagine a node with two pods running: `foo` and `bar` and a disruption dropping all outgoing network packets:

- Applying this disruption at the `pod` level and with a selector targeting the `foo` pod will result with the `foo` pod not being able to send any packets, but the `bar` pod will still be able to send packets, as well as other processes on the node.
- Applying this disruption at the `node` level and with a selector targeting the node itself, both `foo` and `bar` pods won't be able to send network packets anymore, as well as all the other processes running on the node.

## Duration

The `Disruption` spec takes a `duration` field. This field represents amount of time after the disruption's creation before
all chaos pods automatically terminate and the disruption stops injecting new ones. This field takes a string, which is meant to conform to
golang's time.Duration's [string format, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration) This time is measured from the moment
that the Disruption resource is created, not from when the injection of the actual failure occurs. It functions as a strict maximum for the lifetime of the Disruption, not a guarantee of how long the failure will persist for.

If a `duration` is not specified, then a disruption will receive the default duration, which is configured at the controller level by setting
`controller.defaultDuration` in the controller's config map, and this value defaults to 1 hour.

After a disruption's duration expires, the disruption resource will live in k8s for a default of 10 minutes. This can be configured by altering
`controller.expiredDisruptionGCDelay` in the controller's config map.

If any of the options in `spec.triggers` are set, the `duration` will not "begin" until _after_ the injection starts. See [below for details.](#Triggers-aka-controlling-the-timing-of-chaos-pod-creation-and-injection)

## Triggers, aka, Controlling the timing of chaos pod creation and injection

The `Disruption` spec has a `triggers` field, with two subfields: `createPods` and `inject`. These have identical subfields of their own: `notBefore` and `offset`. `spec.triggers` as well as all of its subfields, are strictly optional.
These fields allow you to have more fine-grained control over when chaos pods are created, and when they inject the disruption. After creating a new `Disruption` without setting `spec.triggers`, the chaos-controller will begin creating chaos pods for eligible targets almost immediately, and
those chaos pods, once started, will immediately being injecting the disruption.

Sometimes you don't want the injected failure to occur immediately after the `Disruption` object is created.

`spec.triggers.createPods` controls the timing of chaos pod creation. When set, the chaos-controller will still continue to reconcile your `Disruption` object, but
will not select any targets or create any chaos pods until after the specified time.

`spec.triggers.inject` controls the timing of chaos pod injection. When chaos pods startup, they will first check if the `Disruption` has `spec.triggers.inject` has been set. If yes, they will wait
until the specified timestamp before injecting the failure. However, they will still be responsive to os signals, so the chaos pods or the entire `Disruption` can be deleted. This can allow you to see _exactly_ which chaos pods will be created for which targets, and leave yourself plenty of time
to delete the `Disruption` before any failure injection occurs. This also helps us to, but does not guarantee, deliver simultaneous injection of failure across all targets. The chaos pods do not directly synchronize with each other or with the chaos-controller, so clock skew can be an issue here.

As mentioned above, both options under `spec.triggers`: `spec.triggers.createPods` and `spec.triggers.inject` themselves have two mutually exclusive options:

- `notBefore` takes an RFC3339 formatted timestamp string, eg., "2023-05-09T11:10:08-04:00". This is an absolute value, that must be _after_ the creationTimestamp of the `Disruption`.
- `offset` which takes a golang time.Duration [string, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration). This is a relative value, and thus is more convenient when re-using a `Disruption` yaml.

When `spec.triggers.createPods.offset` is set, the `offset` is measured from the creationTimestamp of the `Disruption`. This allows you to say `spec.triggers.createPods.offset: 5m`, and the chaos pods won't be created until 5 minutes after the Disruption was created.
When `spec.triggers.inject.offset` is set, the `offset` is measured from the timestamp of `spec.triggers.createPods` if defined, and if not, it will be measured from the creationTimestamp of the `Disruption`.

Though the name `notBefore` hopefully implies it, we do need to be explicit that these timestamps are the _earliest_ possible time we may create pods or inject failures. Due to the asynchronous nature of kubernetes controllers, it is probable that pod creation
and thus also injection, could occur after the `notBefore` timestamp.

## Pulse

The `Disruption` spec takes a `pulse` field. It activates the pulsing mode of the disruptions of type `cpu_pressure`, `disk_pressure`, `dns_disruption`, `grpc_disruption` or `network_disruption`. A "pulsing" disruption is one that alternates between an active injected state, and an inactive dormant state. Previously, one would need to manage the Disruption lifecycle by continually re-creating and deleting a Disruption to achieve the same effect.

It is composed of three subfields: `initialDelay`, `dormantDuration` and `activeDuration`, which take a string, which is meant to conform to
golang's time.Duration's [string format, e.g., "45s", "15m30s", "4h30m".](https://pkg.go.dev/time#ParseDuration) and **have to be greater than 500 milliseconds**.

`initialDelay` will specify a duration of a sleep between a chaos pod's startup and the first `activeDuration`. This field is optional.

`dormantDuration` will specify the duration of the disruption being `dormant`, meaning that the disruption will not be injected during that time.

`activeDuration` will specify the duration of the disruption being `active`, meaning that the disruption will be injected during that time.

The pulsing disruption will be injected for a duration of `activeDuration`, then be clean and dormant for a duration of `dormantDuration`, and so on until the end of the disruption.

If a `pulse` is not specified, then a disruption will not be pulsing.

## Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. It's possible to specify multiple label selectors, in which case the controller will select from targets that match all of them. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

See [targeting](./targeting.md)

## Applying a disruption on pod initialization

> :memo: This mode has some restrictions:
>
> - it requires a 1.15+ Kubernetes cluster
> - it requires the `--handler-enabled` flag on the controller container
> - it only works for network related (network and dns) disruptions
> - it only works with the pod level
> - it does not support containers scoping (applying a disruption to only some containers)

It can be handy to disrupt packets on pod initialization, meaning before containers are actually created and started, to test startup dependencies or init containers. You can do this in only two steps:

- redeploy your pod with the specific label `chaos.datadoghq.com/disrupt-on-init` to hold it in the initialization state
  - the chaos-controller will inject an init containers name `chaos-handler` as the first init container in your pod
  - this init container is lightweight and does nothing but waiting for a `SIGUSR1` signal to complete successfully
    - thus, until a disruption targets the pod with the init container, it will do nothing but wait until it times out. The init container has no k8s api access, and does no proactive searching for existing disruption resources.
- apply your disruption [with the init mode on](../examples/on_init.yaml)
  - the chaos pod will inject the disruption and unstuck your pod from the pending state

Note that in this mode, only pending pods with a running `chaos-handler` init container and matching your labels + the special label specified above will be targeted. The `chaos-handler` init container will automatically exit and fail if no signal is received within the specified timeout (default is 1 minute).

## Notifier

When creating a disruption, you may wish to be alerted of important lifecycle warnings (disruption found no target, chaos pod is stuck on removal, target is failing, target is recovering, etc.) through the Notifier module of the chaos-controller. On each occurrence, these events will be propagated through the different set up notifiers (currently `noop/console`, `slack` and `datadog` are implemented).

You can find the complete list of the events sent out by the controller [here](/api/v1beta1/events.go#L24).

Any setup/config error will be logged at controller startup.

### Slack

The `slack` notifier requires a slack API Token to connect to your org's slack workspace.

It will use the disruption's creator username in kubernetes (based on your authentication method) as the email address of the person to send a DM on slack. The DM will come from the slack username 'Disruption Status Bot'. **The email address used to authentify on the kubernetes cluster and create the disruption needs to be the same used on the slack workspace** or the notification will be ignored.

In addition to that, you can receive notifications on the disruption itself by filling the `reporting` field (See the [disruption reporting section](#disruption-reporting)).

### Datadog

The `datadog` notifier requires the `STATSD_URL` environment variable to be set up. It will either send a `Warn` event for warning kubernetes events or a `Success` event for normal recovered kubernetes events sent out by the controller.

### HTTP

The `http` notifier requires a `URL` to send the POST request to and optionally ask for either the list of headers in the configmap or the filepath of a file containing the list of headers to add to the request if needed. It will send a json body containing the notification information.

_Note that the list of headers from the configmap will take prevalence over the list of headers found in the file: if there are conflicting headers in both of those lists, the one from the configmap will be kept._

The list is of format:

```
key1:value1
key2:value2
```

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
    headers: # optional, list of headers to add to the http POST request we send
      - "Authorization:Bearer token"
    headersFilepath: <headers file path> # optional, path to a file containing the list of headers to add to the http POST request we send for the http notifier

```

### Disruption reporting

On top of global notifiers configuration sent to user privately and potentially mirrored to a common slack channel, you may want to send notifications to a dedicated channel on a per disruption basis, to enable your team to be notified of an on-going disruption as an example.

#### How to activate per disruption slack

In order to activate such capability, you will need to:

1. provide the `reporting` field on a disruption spec
2. add the slack bot to your slack workspace
3. [add the slack bot](https://slack.com/help/articles/201980108-Add-people-to-a-channel) to the expected channel(s)
4. configure `chaos-controller` slack notifier with a slack token and enable it

#### Reporting spec example

```yaml
reporting: # optional, add custom notification for this disruption
  slackChannel: team-slack-channel # required, custom slack channel to send notifications to (can be a name or slack channel ID)
  purpose:
    | # required, purpose/contextual informations to explain reasons of the disruption launch, can contain markdown formatting
    *full network drop*: _aims to validate retry capabilities of demo-curl_. Contact #team-test for more informations.
  minNotificationType: Info # optional, minimal notification type to be notified, default is Success, available options are Info, Success, Warning, Error
```
