# Metrics and events

## Controller

* `chaos.controller.nfi.reconcile` is the count of reconcile loop for network failures
* `chaos.controller.nfi.reconcile.duration` is the time passed in the reconcile loop for network failures

## Injector

Events are sent when a new rule is injected and when the rules are cleared.

A bunch of metrics are sent as well:

* `chaos.nfi.injected` is the count of injected network failures
* `chaos.nfi.cleaned` is the count of cleaned network failures
* `chaos.nfi.rules.injected` is the count of injected iptables rules

Every metric has the following common tags:

* `status` which can be `succeed` or `failed` to represent the succeed or the failure of the injection
* `containerid` is the affected container ID
* `uid` is the Kubernetes `NetworkFailureInjection` resource UUID
