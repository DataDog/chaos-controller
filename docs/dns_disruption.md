# DNS disruption

The `dns` field offers a way to inject invalid DNS records:

* `hostname` is a regular expression specifying the hostname(s) to match on
* `record.type` can be set to either "A" or "CNAME", and indicates the type of DNS record to override
* `record.value` should either be a comma-delimited list of IPs if `record.type` is "A" or a url of `record.type` is CNAME. The specified values will be returned on any DNS queries that match `hostname` on the target. If a comma-delimited list of IPs is specified for an A record, they will be used in a round-robin fashion.
