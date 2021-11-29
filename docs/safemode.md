# Safemode

Safemode represents a number of safety nets we have implemented into the chaos controller to help new and experienced users feel more confident deploying new disruptions to their systems. 

The Chaos Controller can be scary to use in production environments, but we learn a lot more running chaos experiments in our production environments. Considering this, having safety nets in place makes the entire process of running chaos experiments in product environments a little safer.

The disruption config as a parameter appropriately named `safe-mode` which represents a boolean value. By default this is turned **off**.

### Safety Nets

| Safety Net  | Description |
| ----------- | ----------- |
| Namespace-wide Targeting                                  | Using generical label selectors (e.x. X) that selects a majority of pods/nodes in a namespace as a target to inject a disruption into                     |
| No Port or Host Specified                                 | Running a network disruption without specifying a port or host                                                                                            |
| Sporadic Targets                                          |  In a volatile environment where targets are being terminated and created sporadically, disruptions should not be allowed to continue disrupting          |
| Editing Spec During Bad Clean up                          | When the controller is having trouble cleaning up, the disruption should not be edited to add/remove targets for example.                                 |
| Multiple CRD Applies on the Same Targets                  | In the case that multiple people are running disruptions against the same set of targets async                                                            |
| Specific Container Disk Disruption on Multi Container Pod | The disk disruption is a pod-wide disruption, if a user tries to specify a specific container, they may be unware they are affecting all other containers |







