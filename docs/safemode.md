# Safemode

Safemode represents a number of safety nets we have implemented into the chaos controller to help new and experienced users feel more confident deploying new disruptions to their environments.

Chaos engineering can be scary to use in production environments, but we have learned a lot after using the chaos-controller for years. We've attempted to coalesce these lessons into safety nets that prevent common dangerous options. Having safety nets in place makes the entire process of running chaos experiments in high value environments a little safer.

Safemode is always enabled by default and will require manual disabling of safety nets in order to bypass. In the disruption schema we have `unsafeMode` which represents ways to remove the safety nets and configure them differently from the default.
`unsafeMode.disableAll` turns off all safety nets. The other options under `unsafeMode` represent individual safety nets which can be disabled independently.
Please take a look at the example below to see how to use `unsafeMode`.

## Ignoring Safety Nets

Because the list of safety nets to be implemented will grow in the future, there will surely be overlap with safety nets which will make it difficult for a user who is confident a specific safety net is not necessary but unsure if others will be.
Therefore the controller allows for you to disable specific safety nets in the Safemode Spec. Checkout out example below to see how to remove certain safety nets.
Keep in mind that all safety nets are turned on by default.

## Configuring Safety Nets

Because each situation will differ for each user, further configuration can be done on each safety net. One example is the safety net `Large Scope Targeting`. 
This safety net has two parameters it checks to determine if the scope of a disruption exceeds acceptable parameters. Those two parameters are the namespace threshold and the cluster threshold.
If the percentage of targets exceeds any of those thresholds, the safety net is caught and the disruption is halted. A user can use the further configuration in order to change these thresholds to what ever percentage makes the most sense to them.
For an example of how to use these configuration, please take a look at the example towards the end of the doc. The following list are configurations currently available for the safety nets:
```yaml
...
unsafeMode:
  config:
    countTooLarge:
      namespaceThreshold: 60 # an integer between 0 - 100 representing a percentage threshold that is acceptable for namespace size percentage
      clusterThreshold: 90 # an integer between 0 - 100 representing a percentage threshold that is acceptable for cluster size percentage
...
```

### Safety Nets

| Safety Net                    | Type | Description                                                                                                                                                             | IgnoreName                 |
|-------------------------------| ----------- |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| Large Scope Targeting         | Generic | Running any disruption with generic label selectors that select a majority of pods/nodes in a namespace as a target to inject a disruption into                         | DisableCountTooLarge       |
| No Port and No Host Specified | Network | Running a network disruption without specifying a port and a host                                                                                                       | DisableNeitherHostNorPort  |


#### Example of Disabling Specific Safety Net

```yaml
# In this disruption we are leaving out host and port but disabling the safety net that catches it so we can continue the disruption
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-ingress
  namespace: chaos-demo
spec:
  level: pod
  selector:
    app: demo-curl
  unsafeMode:
    disableNeitherHostNorPort: true
  count: 1
  network:
    drop: 10
    flow: ingress # disrupt incoming traffic instead of outgoing (requires at least a port or a host to be specified, only works for TCP, please read implementation details before using to know the current limitations)
    hosts:
      drop: 100
```

#### Example of Using Configuration for individual Safety Nets

```yaml
# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2022 Datadog, Inc.

apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-pressure-read
  namespace: chaos-demo
spec:
  level: pod
  selector:
    app: demo-curl
  count: 1
  unsafeMode:
    config:
      countTooLarge:
        namespaceThreshold: 90 # default is 80%
        clusterThreshold: 90 # default is 66%
  diskPressure:
    path: /mnt/data # mount point (in the pod) to apply throttle on
    throttling:
      readBytesPerSec: 1024 # read throttling in bytes per sec
```