# DisruptionCron

## Overview
The `DisruptionCron` is a Custom Resource Definition (CRD) that enables scheduling `Disruptions` against the Kubernetes resources at specified intervals. This tool enhances the process of chaos engineering by providing a means to continually assess a system's resilience against a wide array of potential disruptions, thereby reducing the necessity for manual intervention.

## Why use DisruptionCron?
DisruptionCron facilitates chaos engineering by automating and scheduling chaos experiments. This promotes continual resilience improvement and proactive vulnerability detection, thereby mitigating risks and potential costs associated with unexpected system failures.

## Usage
To schedule a disruption in a cluster, run `kubectl apply -f <disruption_cron_file>.yaml`. To halt the scheduled disruptions, use `kubectl delete -f <disruption_cron_file>.yaml`.

## Example
The following DisruptionCron manifest example triggers node failure disruptions every 15 minutes:
```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: DisruptionCron
metadata:
  name: node-failure
  namespace: chaos-demo # it must be in the same namespace as targeted resources
spec:
  schedule: "*/15 * * * *" # cron syntax specifying that disruption occurs every 15 minutes
  targetResource: # a resource to target
    kind: deployment # a resource name to target
    name: demo-curl # can be either deployment or statefulset
  disruptionTemplate:
    count: 1 # the number of resources to target, can be a percentage if you suffix with "%", e.g. `count: 100%`
    duration: 1h # the amount of time before your disruption automatically terminates itself, for safety
    nodeFailure: # trigger a kernel panic on the target node
      shutdown: false # do not force the node to be kept down
```

## Writing a DisruptionCron spec
### Schedule syntax
The `.spec.schedule` field is required. The value of that field follows the [Cron](https://en.wikipedia.org/wiki/Cron) syntax:
```
# ┌───────────── minute (0 - 59)
# │ ┌───────────── hour (0 - 23)
# │ │ ┌───────────── day of the month (1 - 31)
# │ │ │ ┌───────────── month (1 - 12)
# │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday;
# │ │ │ │ │                                   7 is also Sunday on some systems)
# │ │ │ │ │                                   OR sun, mon, tue, wed, thu, fri, sat
# │ │ │ │ │
# * * * * *
```
For instance, `0 12 * * 5` states that the task must be started every Friday at noon.
To generate CronJob schedule expressions, you can also use web tools like [crontab.guru](https://crontab.guru/).

### Target resource
The `.spec.targetResource` field specifies which resource to run disruptions against, and is required. Since DisruptionCrons are designed to be semi-permanent, they're best used to target other long-lasting resources. As such, the `.spec.targetResource.kind` field can only be set to either `deployment` or `statefulset`. At runtime, pods from either of these resources are randomly selected for disruption.

### Disruption template
The `.spec.disruptionTemplate` defines a template for the Disruptions that the DisruptionCron creates, and it is required.
Its schema is identical to a `DisruptionSpec` of the `Disruption` CRD. [Detailed examples](examples.md) of various Disruption types, including their respective manifests, are readily available for reference.

### Tolerance for delayed disruption start
The `.spec.delayedStartTolerance`  field is optional. It establishes a time threshold for starting the Disruption. It accepts values in the format of golang's `time.Duration`, like "45s", "15m30s", or "4h30m".

If a Disruption doesn't start on time and goes beyond this threshold, that particular instance of the Disruption is skipped but future occurrences remain scheduled. When there's no specified `delayedStartTolerance`, there's no time limit, and Disruptions can begin after any delay.

For instance, with a setting of "200s", the Disruption can begin up to 200 seconds past its scheduled time."