# Safemode

Safemode represents a number of safety nets we have implemented into the chaos controller to help new and experienced users feel more confident deploying new disruptions to their environments.

Chaos engineering can be scary to use in production environments, but we have learned a lot after using the chaos-controller for years. We've attempted to coalesce these lessons into safety nets that prevent common dangerous options.Having safety nets in place makes the entire process of running chaos experiments in high value environments a little safer.

Safemode is always enabled by default and will require manual disabling of safety nets in order to bypass. In the disruption schema we have `unsafeMode` which represents ways to remove the safety nets.
`unsafeMode.disableAll` turns off all safety nets. The other options under `unsafeMode` represent individual safety nets which can be disabled independently.
Please take a look at the example below to see how to use `unsafeMode`.

## Ignoring Safety Nets

Because the list of safety nets to be implemented will grow in the future, there will surely be overlap with safety nets which will make it difficult for a user who is confident a specific safety net is not necessary but unsure if others will be.
Therefore the controller allows for you to disable specific safety nets in the Safemode Spec. Checkout out example below to see how to remove certain safety nets.
Keep in mind that all safety nets are turned on by default.

### Safety Nets

| Safety Net                                                | Type | Description                                                                                                                                                            | IgnoreName                 |
|-----------------------------------------------------------| ----------- |------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| Namespace-wide Targeting                                  | Generic | Running any disruption with generic label selectors that select a majority of pods/nodes in a namespace as a target to inject a disruption into                        | DisableCountTooLarge       |
| Sporadic Targets                                          | Generic | Running any disruption in a volatile environment where targets are being terminated and created sporadically, disruptions should not be allowed to continue disrupting | DisableSporadicTargets     |
| No Port and No Host Specified                             | Network | Running a network disruption without specifying a port and a host                                                                                                      | DisableNeitherHostNorPort  |
| Specific Container Disk Disruption on Multi Container Pod | Disk    | Running a disk disruption, if a user tries to specify a specific container, they may be unaware they are affecting all other containers                                | DisableSpecificContainDisk |


#### Example of Disabling Safemode

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