# Metrics and events

Here's the list of metrics sent by the controller and the injector.

## Controller

* `chaos.controller.restart` increments when the controller is restarted
* `chaos.controller.reconcile` increments when the reconcile loop is called
* `chaos.controller.reconcile.duration` is the time passed in the reconcile loop
* `chaos.controller.inject.duration` is the time it took to fully inject the disruption since its creation
* `chaos.controller.cleanup.duration` is the time it took to fully cleanup the disruption since its deletion
* `chaos.controller.pods.created` increments when a chaos pod is created
* `chaos.controller.pods.gauge` is the total count of existing chaos pods
* `chaos.controller.disruption.completed_duration` is the complete life time of the disruption, from creation to deletion
* `chaos.controller.disruption.ongoing_duration` is the duration of the disruption so far, from creation to now
* `chaos.controller.disruptions.stuck_on_removal` increments when a disruption is stuck on removal
* `chaos.controller.disruptions.stuck_on_removal_total` is the total count of existing disruption being flagged as stuck on removal
* `chaos.controller.disruptions.gauge` is the total count of existing disruption
* `chaos.controller.disruptions.count` increments when a disruption is created

### Admission webhooks

* `chaos.controller.validation.failed` increments when a disruption fails to be validated from the admission webhook
* `chaos.controller.validation.created` increments when a disruption is created
* `chaos.controller.validation.updated` increments when a disruption is updated
* `chaos.controller.validation.deleted` increments when a disruption is deleted

## Injector

* `chaos.injector.injected` increments when a disruption is injected
* `chaos.injector.cleaned` increments when a disruption is cleaned
