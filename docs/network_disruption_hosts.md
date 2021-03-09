# Network disruption: Specifying hosts

## Q: When should I specify hosts?

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/notation_traffic.png" height=180 width=600 />
</kbd></p>

As with all disruptions, pods or nodes are targeted for injection if they satisfy the conditions of the label selector specified in the `selector` field. For network disruptions, we can also specify to only disrupt packets interacting with a particular host or set of hosts through the `hosts` field.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/notation_egress.png" height=160 width=570 />
</kbd></p>

## Notation

Although the `hosts` field is handled the same between pod-level and node-level, the way we target network interfaces varies based on how nodes are configured. For example, whether pods have their own networking interface or use its host's networking directly affects which interface gets targeted:

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/interfaces_node_pod.png" height=290 width=550 />
</kbd></p>

For diagrams in this documentation, arrows represent traffic flow from a particular interface to another. They do not represent the entire path a packet takes to arrive at another pod unless an intermediate device or packet alteration affects how the disruption applies.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/notation_targets.png" height=100 width=570 />
</kbd></p>

Note also that the flow (`ingress` vs `egress`) for a disruption should be selected deliberately based on [this documentation](/docs/network_disruption_flow.md).

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/notation_ingress.png" height=160 width=570 />
</kbd></p>

Let's see some examples of different use cases!

### Assumptions

We assume that packets identified as necessary for healthchecks from the cloud service provider or for communications with Kubernetes are ignored.

Additionally, note that the `hosts` field expects a list where the items of the list need not be of the same time (for example, you can have a hostname, IP address, and CIDR block in one disruption). They are visualized separately below to avoid confusion.

See the **Some special cases** section for examples of specifying the `port` and `protocol` fields.

## Pod Level Examples

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_no_disruption.png" height=240 width=550 />
</kbd></p>

Pods typically have a single interface with which it interacts with the outside world. Let's take this set of pods from various applications and see how different `hosts` field configurations affect the traffic differently.

### Case 1: No host is specified

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_no_hosts_egress.png" height=330 width=600/>
</kbd></p>

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_no_hosts_ingress.png" height=330 width=600/>
</kbd></p>

If no `hosts` field is specified, all packets aside from those explicitly whitelisted in the **Assumptions** will be disrupted, in this case dropping 50% of traffic leaving (`egress`) or entering (`ingress`) the interface.

Note: `ingress` traffic disruption is only guaranteed for TCP, not UDP. See [this documentation](/docs/network_disruption_flow.md) for more details. For the remainder of this documentation, we will only discuss the default flow configuration (`egress`).

### Case 2: IP address specified

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_ip.png" height=330 width=600/>
</kbd></p>

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_ips.png" height=330 width=600/>
</kbd></p>

If the `hosts` field contains one or multiple IP addresses, the routing table is consulted, and interfaces containing routes to those IPs will get targeted. In the absence of such interface, the default interface gets selected. In the case of pods, there is usually one interface, `eth0`.

### Case 3: CIDR block specified

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_cidr.png" height=330 width=600/>
</kbd></p>

If the `hosts` field contains one or multiple CIDR blocks, the routing table is consulted. If the list of IP addresses in the CIDR block overlap with IP addresses which comprise the route entries of an interface, the interface is targeted. In the absence of such interface, the default interface gets selected.

Note: Only use CIDR blocks with `egress` traffic. 

### Case 4: Hostname specified

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_hostname.png" height=330 width=600/>
</kbd></p>

### Some special cases

Cluster IPs can also be specified to target the relevant pods.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_cluster_ip.png" height=330 width=600/>
</kbd></p>

All of these use cases may have `port` configurations, `protocol` configurations, or both can be applied on top of the studied disruption. Packets going through the same interface which do not meet those criteria will be delivered as if they were not targeted (some nuance on this in the [tc documentation](/docs/network_disruption_prio.md)).

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_protocol_port_egress.png" height=330 width=600/>
</kbd></p>

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/pods_protocol_port_ingress.png" height=330 width=600/>
</kbd></p>

## Node Level Examples

Pods usually come with one network interface. This is common for nodes as well, but large nodes can have multiple interfaces to which pods are assigned one-to-many.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/nodes_interface_simple.png" height=250 width=320/>
    <img src="../docs/img/network_hosts/nodes_interface_complex.png" height=250 width=320/>
</kbd></p>

### Case 1: Label selector for a few nodes

When a node is targeted, all interfaces with route table entries to IP addresses specified in `hosts` (for completeness, this can be through an explicit list of IP addresses, CIDR blocks, hostnames, or some combination of these) will be targeted. Exactly as with pod leve, a `tc` rule is applied which impacts packets with destination IPs satisfying these `hosts` criteria.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/nodes_label_small.png" height=330 width=600/>
</kbd></p>

Keep in mind that the diagrams thus far represent that all network interfaces for have a routing entry to any pod we wish to .

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/nodes_limited_routes.png" height=330 width=600/>
</kbd></p>

### Case 2: Disrupting an entire AZ
If we disrupt an entire Availability Zone, note that the pods encompassed by the `hosts` field often fall within the nodes targeted.

<p align="center"><kbd>
    <img src="../docs/img/network_hosts/nodes_label_az.png" height=330 width=600/>
</kbd></p>
