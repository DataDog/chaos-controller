# Handling changes in the environment of the targets

## Network disruption: Dynamic service resolution

It is possible for network disruptions to only filter on the traffic going to specific kubernetes services:

```
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-drop
  namespace: chaos-demo
spec:
  level: pod
  selector:
    app: demo-curl
  count: 1
  network:
    drop: 100 # percentage of outgoing packets to drop
    *ervices: # optional, list of destination Kubernetes services to filter on
      - name: demo # service name
        namespace: chaos-demo # service namespace
```

In order to have an up to date state of the filtered on services / pods destination, we dynamically resolve them by:
* installing **kubernetes watchers** on the kubernetes services and kubernetes pods (more info on watchers [here](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes))
* keeping track of **tc filters** (more info on filters [here](https://man7.org/linux/man-pages/man8/tc-u32.8.html))

### TC Filters Technicalities

To delete tc filters, we need to keep in memory the priority (or preference) of each tc filter created, by assigning a priority when adding a tc filter:

`tc filter add dev eth0 priority 49155 parent 1:0 u32 match ip dst 10.98.115.140/32 match ip dport 8080 0xffff match ip protocol 6 0xff flowid 1:4`

The priority will indicate the order of the filter. (We don't need for the filters to have a specific order at this stage)

A filter can then be deleted:

`tc filter delete dev eth0 priority 49155`

In the case of a service filtered on getting deleted by the user, the tc filters on the related pods will not be deleted; the pods itself are not modified by a change in the service.