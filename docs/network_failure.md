# NetworkFailureInjection ([example](../config/samples/chaos_v1beta1_networkfailureinjection.yaml))

The `NetworkFailureInjection` resource provides an automated way of dropping the connection between a pod and a service. Please note that the connection is dropped when outgoing from the pod you targeted. It means you can prevent the targeted pod from querying an API but not from being queried. However, if the call to query to targeted pod is using TCP, the SYN-ACK answer to establish the connection will never be sent and the result will be quite the same.

## Protocol

The protocol can be either `tcp` or `udp`.

## Host (optional)

The host field can be either a single IP, an IP block (CIDR) or a hostname.

If a hostname is provided, it'll be resolved as it would be resolved in the targeted pod. All IPs contained in the record will be included in the connection drop (useful if the underlying service uses DNS RR load balancing).

If the host field is not specified, then the connection will be dropped for all hosts, no matter the IP (`0.0.0.0/0`).

## Probability (optional)

The probability field is an integer between 1 and 100 and is the percentage of packets to drop.

If not specified (or set to 0), all the packets will be dropped (eq. to a probability of 100%).

## numPodsToTarget (optional)

The amount of pods to target. If this number isn't specified or is greater than the actual number of pods targeted by the label selector, all pods will be selected.

This selection is random.
