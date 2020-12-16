# Metrics and events

## Controller

* `chaos.controller.reconcile` is the count of reconcile loops
* `chaos.controller.reconcile.duration` is the time passed in the reconcile loop
* `chaos.controller.injection.duration` is the time it took to create injection pods since the creation of the disruption resource
* `chaos.controller.cleanup.duration` is the time it took to create cleanup pods since the deletion of the disruption resource
* `chaos.controller.pods.created` is the amount of pods created for injection and cleanup phases (can be filtered with the `phase` tag)

## Injector

Events are sent when a new rule is injected and when the rules are cleared.

A bunch of metrics are sent as well:

* `chaos.nfi.injected` is the count of injected network failures
* `chaos.nfi.cleaned` is the count of cleaned network failures

Every metric has the following common tags:

* `status` which can be `succeed` or `failed` to represent the succeed or the failure of the injection
