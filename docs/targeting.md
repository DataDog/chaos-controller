# Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. It's possible to specify multiple label selectors, in which case the controller will select from targets that match all of them. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

The default behavior for a disruption is Dynamic Targeting. Please use the `StaticTargeting` configuration flag in your disruption ([see example](../examples/static_targeting.yaml)) if you wish to deactivate it. [Read StaticTargeting](<#Static Targeting>).

## Dynamic Targeting (default)

By default, there is a constant re-targeting for disruptions in time. This means at any given time, a valid target within the selector's scope will be added to the target list and be disrupted.

> :warning: :warning:  Although activated by default, this feature is to use with care as a disruption gone wrong can get out of control: per example, a disruption targeting 100% of an application's pod will affect all existing **and** future pods which can appear once the disruption started. As long as this 100% disruption exists, there will be no spared pod. 

Dynamic Targeting behavior design choices:

- the controller will reconcile/update its targets list on any chaos or selector pod movement (create, update, delete)
- the controller will consider as a still-alive target any pod/node that exists - regardless of its state.

For more information on Dynamic Targeting implementation details, please check [Dynamic Targeting Technicals](<#NB: Dynamic Targeting Technicals>)

## Static Targeting

Activate `StaticTargeting` to limit the disruption to a single target selection step at the disruption's creation. It allows for more controlled disruption impact and propagation, as the targets will never change and _can_ be compensated for in case they are made useless. Its major limit is not being able to follow targets through deployments/rollouts.

See provided [example](../examples/static_targeting.yaml).

## Targeting safeguards

When enabled [in the configuration](../chart/values.yaml) (`controller.enableSafeguards` field), safeguards will exclude some targets from the selection to avoid unexpected issues:

- if the disruption is applied at the node level, the node where the controller is running on can't be selected
- if the disruption is applied at the pod level with a node disruption, the node where the controller is running on can't be selected

## Advanced targeting

In addition to the simple `selector` field matching an exact key/value label, one can do some more advanced targeting with the `advancedSelector` field. It uses the [label selector requirements mechanism](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#LabelSelectorRequirement) allowing to match labels with the following operator:

- `Exists`: the label with the specified key is present, no matter the value
- `DoesNotExist`: the label with the specified key is not present
- `In`: the label with the specified key has a value strictly equal to one of the given values
- `NotIn`: the label with the specified key has a value not matching any of the given values

You can look at [an example of the expected format](../examples/advanced_selector.yaml) to know how to use it.

### Filtering

The `selector` and `advancedSelector` fields only use kubernetes resource labels, as the kubernetes api only allows for listing resources based on their labels. However, it's perfectly valid to want to filter your targets to those containing specific annotations. The `spec.filter` field currently has a single subfield, `spec.filter.annotation`, which takes a set of key/value pairs. It works similarly to the `selector` field, in that all targets must have annotations matching _all_ specified k/v pairs. We will filter targets initially based on the label selectors used, before applying any filters.

## Targeting a specific pod

How can you target a specific pod by name, if it doesn't have a unique label selector you can use? The `Disruption` spec doesn't support field selectors at this time, so selecting by name isn't possible. However, you can use the `kubectl label pods` command, e.g., `kubectl label pods $podname unique-label-for-this-disruption=target-me` to dynamically add a unique label to the pod, which you can use as your label selector in the `Disruption` spec.

## Targeting a specific container within a pod

By default, a disruption affects all containers within the pod. You can restrict the scope of the disruption to a single container or to only some containers [like this](../examples/containers_targeting.yaml).

---

### NB: Dynamic Targeting Technicals

![Dynamic Targeting Diagram](https://user-images.githubusercontent.com/17198797/157055173-f4ab9d94-5c4d-419d-a08f-160fd41c5f23.png)

### Case for the use of kubernetes cache in dynamic targeting

#### When and how should we run the reconcile loop with dynamic targeting ?

In order to make dynamic targeting work, we needed to run the reconcile loop more often then we previously did; the bread-and-butter case of Dynamic Targeting is having new targets added to an existing disruption. 

As trivial as it sounds, new targetable pods are not yet registered and monitored by the controller; it needs to run the reconcile loop to take them into account. So how do we make the controller run the reconcile loop for objects it possibly does not even know about ? A few terrible solutions instantly come to mind:

- Run the reconcile loop periodically for no reason to scan for new pods matching the disruption's selector :no_entry:

  Would work, but would not scale well -as one controller can start running more and more disruptions simultaneously-, waste resources running the loop for sometimes no reason, and input an arbitrary time scale. **We should not trigger the loop for no reason and need external signals.**

- Run the reconcile loop on any cluster activity (pod/node CREATE, UPDATE, DELETE events) :no_entry:

  Would be very resource-expensive, and would trigger all disruptions' reconcile loops at the same time. The controller would be constantly running all its' disruptions reconcile loops. **We cannot afford to trigger the loop on all possible external signals; we need to filter them.**

In the end, it's all about reconciling the relevant disruptions as a reaction to the relevant events: chaos-pods, targets and potential targets CRUD operations.

#### So how do we filter kubernetes signals correctly ?

Each disruption contains its given selector, describing a set of desired targets specific to that disruption. For each disruption, the relevant signals to pick up are those which belong to objects fitting its selector. On signals from fitting objects, we need to trigger/enqueue a reconcile for the given disruption.

To solve those problems, we use a client-go kubernetes object called a [cache](https://pkg.go.dev/k8s.io/client-go/tools/cache) to which is attached an Informer object, and an event handler. They serve the purpose of an Observer.

For each new disruption the controller receives, the controller will create a [new cache object](https://github.com/DataDog/chaos-controller/blob/adb8070c989a6c25195354e3bcef3f2c839ef032/controllers/cache_handler.go#L497), taking the given selector as input. All caches object are stored in a map, using a hash of the disruption's configuration as key for uniqueness, and a cache's lifecycle it correlated to the one of its disruption. On relevant events (and relevant events only), the cache will trigger/enqueue a reconcile loop in the controller specifically for the corresponding disruption.

#### Cache deletion

Using a dedicated cache object for each disruption works great; the reconcile loop is ran on every possible target events, and the disruption adapts wonderfully to changing targets. Resource consumption is stable and seems to scale. However when the disruption is deleted, we need to make sure the cache is deleted as well. This proved more difficult than it looked.

When created, the cache has a context, and a cancel function. Those are stored with the cache reference in the aforementioned map, and called upon the final disruption reconcile. However, if the deletion does not go through for any given reason, this cache can be very hard to recover without restarting the controller. To uphold our safety and performance standards, we need safety mechanisms to assert a cache cannot be lost in nature after a disruption is deleted.

There are three folds:

- [Clear Instance Selector Cache](https://github.com/DataDog/chaos-controller/blob/adb8070c989a6c25195354e3bcef3f2c839ef032/controllers/cache_handler.go#L552): This function is called as late as possible before the disruption deletion. It calls the cache cancel function and deletes the cache map entry. This will cover most cases, and following listings are extra safeties -- because the reconcile loop can be interrupted.
- [Expired Cache Context Cleanup](https://github.com/DataDog/chaos-controller/blob/adb8070c989a6c25195354e3bcef3f2c839ef032/controllers/cache_handler.go#L571): on every reconcile, asserts for every existing cache the corresponding disruption exists. Also, asserts no cache listed in the map has with a already-canceled context - in which case it's deleted.
- [Cache Deletion Safety](https://github.com/DataDog/chaos-controller/blob/adb8070c989a6c25195354e3bcef3f2c839ef032/controllers/cache_handler.go#L596): the first two methods are useful and will prevent most cases from going wrong, but still rely on the controller looping after the disruption is over to make sure the resources are released. Started in its own goroutine this function will, every minute, poll the disruption. If it is not found the cache context will be canceled, releasing the resources it used. If the controller reconciles again, the second listed function will erase the cache from the map, as the context will be found canceled.

To ensure no cache is running loose, a gauge indicating the amount of current cache entries is set to be monitored; it can be subtracted from the gauge of live disruptions to alert in case of sustained negative results.
