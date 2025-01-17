# Network disruption: Specifying cloud managed services

## Why

Large cloud services providers are using wide IP ranges. Hostnames used to identify those services are resolving with some IPs of that range, and resolved IPs can change between each DNS request. Applying a network disruption using those hostnames only doesn’t work well since retrying the resolution of such hostname would return new IPs (not disrupted) and the disruption would be ineffective.

Available providers are:
- AWS
- GCP
- Datadog

### Cloud Provider Manager

The manager handles regular IP ranges files pulls to refresh the file content on enabled providers. The chart contains the default public URL for each cloud provider but it can be customized if needed. You can also disable some providers or all of them in case you want to disable the feature. Please note that if you specify a custom URL, the file must match the same format as the public one to be loaded successfully, otherwise the controller will crash.

```
cloudProviders:
    disableAll: false
    pullInterval: "24h"
    aws:
      enabled: true
      ipRangesURL: "https://ip-ranges.amazonaws.com/ip-ranges.json"
    gcp:
      enabled: true
      ipRangesURL: "https://www.gstatic.com/ipranges/goog.json"
    datadog:
      enabled: true
      ipRangesURL: "https://ip-ranges.datadoghq.com/"
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
        - service: "S3"
          flow: egress # optional, available are egress or ingress
          protocol: tcp # optional, available are tcp or udp
          connState: new # optional, connection state (new: new connections, est: established connections, defaults to all states)
      gcp:
        - service: "Google" # only service available for gcp
          flow: egress # optional, available are egress or ingress
          protocol: tcp # optional, available are tcp or udp
          connState: new # optional, connection state (new: new connections, est: established connections, defaults to all states)
      datadog:
        - service: "api"
          flow: egress # optional, available are egress or ingress
          protocol: tcp # optional, available are tcp or udp
          connState: new # optional, connection state (new: new connections, est: established connections, defaults to all states)
    delay: 1000 # delay (in milliseconds) to add to outgoing packets, 10% of jitter will be added by default
```

## AWS

Available services are:
```
 DYNAMODB, ROUTE53, ROUTE53_RESOLVER, EBS, CODEBUILD, API_GATEWAY, WORKSPACES_GATEWAYS, EC2_INSTANCE_CONNECT, CHIME_VOICECONNECTOR, GLOBALACCELERATOR, CHIME_MEETINGS, CLOUDFRONT_ORIGIN_FACING, AMAZON_APPFLOW, KINESIS_VIDEO_STREAMS, EC2, CLOUDFRONT, ROUTE53_HEALTHCHECKS_PUBLISHING, CLOUD9, ROUTE53_HEALTHCHECKS, S3, AMAZON_CONNECT
```

We do not support using the service "AMAZON" (from the ip ranges file) as it's a combination of all ip ranges from all services and more miscellaneous ips; the number of ip ranges being too much from this, it's not possible for us to filter all of them at once.

We are using the URL **https://ip-ranges.amazonaws.com/ip-ranges.json** to pull all the IP Ranges of AWS.

## GCP

Available service is `Google`.

Google does not indicate which ip ranges correspond to which service in its ip ranges files.

We are using the URL **https://www.gstatic.com/ipranges/goog.json**. This file is the generic Google ip ranges file. We could not use the Google Cloud specific file due to some ip ranges from the apis being in the first file (goog.json). ([More info here](https://support.google.com/a/answer/10026322?hl=en))

We'd like to include the private ranges alongside the public ranges. The private ranges don't appear to be published in a static json file, but are listed in documentation in various places:
https://cloud.google.com/vpc/docs/configure-private-google-access#config-options
https://cloud.google.com/vpc/docs/subnets#restricted-ranges

So we configure this directly in the configmap under `controller.cloudProviders.gcp.extraIpRanges`, which takes a list of strings,
of the form `"service;iprange;iprange;...;iprange`. We aren't able to use a map because of how viper normalizes map keys.

### Datadog

Available services are:
```
 synthetics, orchestrator, process, global, logs, synthetics-private-locations, apm, webhooks, agents, api
```

We are using the URL **https://ip-ranges.datadoghq.com** to pull all the IP Ranges of Datadog.
