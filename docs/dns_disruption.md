# DNS disruption

The `dns` field offers a way to inject invalid DNS records:

* `hostname` is a regular expression specifying the hostname(s) to match on
* `record.type` can be set to either "A" or "CNAME", and indicates the type of DNS record to override
* `record.value` should either be a comma-delimited list of IPs or "NXDOMAIN" if `record.type` is "A". A url should be used if `record.type` is CNAME. The specified values will be returned on any DNS queries that match `hostname` on the target. If a comma-delimited list of IPs is specified for an A record, they will be used in a round-robin fashion.

## How does it work?

In order to ensure the target receives the configured records from DNS queries, the injector takes two steps.

First, it sets up and runs a man-in-the-middle DNS resolver on the chaos pod, which you can find at `./bin/injector/dns_disruption_resolver.py`. This resolver intercepts DNS queries, checks the queried hostname against a local config file, and returns any present record overrides. If the resolver has no matching record for the hostname, it proxies the DNS query to the normal DNS resolver configured for the chaos pod.

Second, in order for the target's DNS queries to end up at the injector's DNS resolver instead of the intended resolver, we use `iptables` nat rules.
With the OnInit parameter, we target all port 53 udp traffic, which is then redirected to the chaos pod, rather than the intended destination. (**It is not possible to isolate containers**)
Without the OnInit parameter, we target all port 53 udp traffic **of each container targeted in the pod**, which is then redirected to the chaos pod, rather than the intended destination.

## Forwarding non-matched requests

Depending on your DNS setup you might need to override the DNS server and/or instruct the controller to forward requests to kube-dns. See the `dnsDisruption` section of the [helm chart](../chart/values.yaml).

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

---

:warning: If the disruption is injected at the pod level, you must enter the pod network namespace first.

* Identify the container IDs of your pod

```
kubectl get -ojson pod demo-curl-547bb9c686-57484 | jq '.status.containerStatuses[].containerID'
"containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460"
"containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d"
```

* Find one of the container PID

```
# crictl inspect cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460 | grep pid
    "pid": 5607,
            "pid": 1
            "type": "pid"
```

* Enter the network namespace

```
# nsenter --net=/proc/5607/ns/net
```

---

* Remove iptables rules jumping to the `CHAOS-DNS` chain

```
# iptables-save | grep -- '-j CHAOS-DNS'
-A POSTROUTING -p udp -m cgroup --cgroup 1048592 -m udp --dport 53 -j CHAOS-DNS
# iptables -t nat -D POSTROUTING -p udp -m cgroup --cgroup 1048592 -m udp --dport 53 -j CHAOS-DNS
```

* Remove iptables `CHAOS-DNS` chain

```
# iptables -t nat -F CHAOS-DNS
# iptables -t nat -X CHAOS-DNS
```

---

:warning: If the disruption is injected at the pod level, you must find the related cgroups path **for each container**.

* Identify the container IDs of your pod

```
kubectl get -ojson pod demo-curl-547bb9c686-57484 | jq '.status.containerStatuses[].containerID'
"containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460"
"containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d"
```

* Identify cgroups path

```
# crictl inspect cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460 | grep cgroupsPath
        "cgroupsPath": "/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460",
```

---

* Reset the `net_cls` value for each container

```
# echo 0 > /sys/fs/cgroup/net_cls/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/net_cls.classid
```

* Kill dns resolver process if it is still running

```
# ps ax | grep dns_disruption_resolver
1150251 ?        S      0:02 /usr/bin/python3 /usr/local/bin/dns_disruption_resolver.py -c /tmp/dns.conf --dns 8.8.8.8 --kube-dns off
1167110 pts/0    S+     0:00 grep dns_disruption_resolver
# kill 1150251
```
