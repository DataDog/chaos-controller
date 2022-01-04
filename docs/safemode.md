# Safemode

Safemode represents a number of safety nets we have implemented into the chaos controller to help new and experienced users feel more confident deploying new disruptions to their environments.

The Chaos Controller can be scary to use in production environments, but we learn a lot more running chaos experiments in our production environments. Considering this, having safety nets in place makes the entire process of running chaos experiments in high value environments a little safer.

The disruption config has a parameter appropriately named `safeMode` which represents a spec containing several boolean fields.
One of the fields, `IgnoreAll` overrides the rest if set to true. It will ignore every safety net.
We recommend that `IgnoreAll` be set on the cluster level by default, so maintainers of the controller can turn it on by default in production and maybe off in staging environments.
When `IgnoreAll` is turned off, the user will have the ability to turn specific safety nets off if they are confident that specific ones are not needed for their use case.
Please take a look at the example below to see how to format Safemode in your next disruption.

## Ignoring Safety Nets

Because the list of safety nets to be implemented will grow in the future, there will surely be overlap with safety nets which will make it difficult for a user who is confident a specific safety net is not necessary but unsure if others will be.
Therefore the controller allows for you to ignore specific safety nets in the Safemode Spec. Checkout out example below to see how to remove certain safety nets.
Remember that these fields to turn off specific safety nets is only accessible to the user if the `IgnoreAll` field is false. Otherwise it overrides all other safety nets and turns them off.
Keep in mind that all safety nets are turned on by default when safemode is on (`IgnoreAll` set to False), so all that is necessary to ignore safety nets.

### Safety Nets

| Safety Net  | Type | Description | IgnoreName |
| ----------- | ----------- | ----------- | ----------- |
| Namespace-wide Targeting                                  | Generic |  Using generical label selectors (e.x. X) that selects a majority of pods/nodes in a namespace as a target to inject a disruption into                     | ignoreCountToLarge        |
| Sporadic Targets                                          | Generic |  In a volatile environment where targets are being terminated and created sporadically, disruptions should not be allowed to continue disrupting           | ignoreSporadicTargets     |
| No Port or Host Specified                                 | Network |  Running a network disruption without specifying a port or host                                                                                            | ignoreNoPortOrHost        |
| Specific Container Disk Disruption on Multi Container Pod | Disk    |  The disk disruption is a pod-wide disruption, if a user tries to specify a specific container, they may be unware they are affecting all other containers | ignoreSpecificContainDisk |


#### Example

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-ingress
  namespace: chaos-demo
spec:
  level: pod
  selector:
    app: demo-curl
  safeMode:
    ignoreAll: false
    ignoreNoPortOrHost: true
  count: 1
  network:
    drop: 10
    flow: ingress # disrupt incoming traffic instead of outgoing (requires at least a port or a host to be specified, only works for TCP, please read implementation details before using to know the current limitations)
    hosts:
      - port: 80
```







