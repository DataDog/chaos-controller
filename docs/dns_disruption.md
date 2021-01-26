# DNS disruption

The `dns` field offers a way to inject invalid DNS records:

* `hostname` is a regular expression specifying the hostname(s) to match on
* `record.type` can be set to either "A" or "CNAME", and indicates the type of DNS record to override
* `record.value` should either be a comma-delimited list of IPs if `record.type` is "A" or a url of `record.type` is CNAME. The specified values will be returned on any DNS queries that match `hostname` on the target. If a comma-delimited list of IPs is specified for an A record, they will be used in a round-robin fashion.

## How does it work?

In order to ensure the target receives the configured records from DNS queries, the injector takes two steps.

First, it sets up and runs a man-in-the-middle DNS resolver on the chaos pod, which you can find at `./bin/injector/dns_disruption_resolver.py`. This resolver intercepts DNS queries, checks the queried hostname against a local config file, and returns any present record overrides. If the resolver has no matching record for the hostname, it proxies the DNS query to the normal DNS resolver configured for the chaos pod.

Second, in order for the target's DNS queries to end up at the injector's DNS resolver instead of the intended resolver, we use `iptables` nat rules. All port 53 udp traffic is redirected to the chaos pod, rather than the intended destination.
