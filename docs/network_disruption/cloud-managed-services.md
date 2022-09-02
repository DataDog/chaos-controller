# Network disruption: Specifying cloud managed services

## Why

Large cloud services providers are using wide IP ranges. Hostnames used to identify those services are resolving with some IPs of that range, and resolved IPs can change between each DNS request. Applying a network disruption using those hostnames only doesn’t work well since retrying the resolution of such hostname would return new IPs (not disrupted) and the disruption would be ineffective.

Available cloud providers are:
- AWS

### Process


### Cloud Provider Manager

The service will pull and parse the IP Ranges from the available cloud providers every x minutes/hours, defined in the chaos-controller configuration:

```
cloudproviders:
    pullInterval: "1d"
```

On the creation of the chaos pod, the chaos-controller will then use those ip ranges for the Network Disruption and transform it into a Host Network Disruption.

### Example


```
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-cloud
  namespace: chaos-demo
spec:
  level: pod
  selector:
    app: demo-cirl
  count: 1
  network:
    cloud:
      aws:
        - "S3"
    delay: 1000 # delay (in milliseconds) to add to outgoing packets, 10% of jitter will be added by default
    delayJitter: 5 # (optional) add X % (1-100) of delay as jitter to delay (+- X% ms to original delay), defaults to 10%
```

