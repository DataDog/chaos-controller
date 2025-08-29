# Pod Replacement

Pod replacement is a chaos engineering disruption that simulates the complete replacement of a Kubernetes pod by cordoning the node hosting the target pod and then deleting the pod to force it to reschedule. This disruption can optionally delete PersistentVolumeClaims (PVCs) associated with the pod to simulate complete storage loss.

## Overview

The pod replacement disruption performs the following steps:
1. **Cordon the node** - Marks the node as unschedulable to prevent new pods from being scheduled
2. **Delete PVCs** (optional) - Removes PersistentVolumeClaims associated with the target pod
3. **Delete the target pod** - Terminates the pod, forcing it to reschedule elsewhere
4. **Uncordon the node** - Marks the node as schedulable again to allow new pods to be scheduled

## Configuration

Pod replacement disruption is configured using the `podReplacement` field in the disruption spec:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: pod-replacement-example
  namespace: chaos-demo
spec:
  level: pod # Must be set to 'pod' for pod-level targeting
  selector:
    app: my-application
  count: 1
  podReplacement:
    deleteStorage: true        # Delete PVCs associated with the pod (default: true)
    forceDelete: false         # Force delete with grace period 0 (default: false)
    gracePeriodSeconds: 30     # Grace period for pod deletion (optional)
```

### Configuration Options

- **`deleteStorage`** (boolean, default: `true`): Determines whether PersistentVolumeClaims associated with the target pod should be deleted. When enabled, this simulates complete storage loss scenarios.

- **`forceDelete`** (boolean, default: `false`): Forces deletion of stuck pods by setting the grace period to 0. Use with caution as this bypasses graceful shutdown procedures.

- **`gracePeriodSeconds`** (integer, optional): Specifies the grace period for pod deletion in seconds. If not specified, uses the pod's default grace period. This setting is ignored when `forceDelete` is true.

## Usage Examples

### Basic Pod Replacement

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: basic-pod-replacement
  namespace: default
spec:
  level: pod
  selector:
    app: web-service
  count: 1
  maxRuns: 1
  podReplacement:
    deleteStorage: true
```

### Pod Replacement Without Storage Deletion

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: pod-replacement-keep-storage
  namespace: default
spec:
  level: pod
  selector:
    app: database
  count: 1
  podReplacement:
    deleteStorage: false  # Keep PVCs intact
    gracePeriodSeconds: 60
```

### Force Delete Pod Replacement

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: force-pod-replacement
  namespace: default
spec:
  level: pod
  selector:
    app: stuck-pod
  count: 1
  podReplacement:
    forceDelete: true  # Immediate termination
```

## Behavior and Considerations

### Node Cordoning
- The disruption cordons (marks as unschedulable) the node hosting the target pod
- This prevents new pods from being scheduled on that node during the disruption
- The node is automatically uncordoned during cleanup when the disruption ends
- If the node is already cordoned, the disruption will not uncordon it during cleanup

### Storage Deletion
- When `deleteStorage` is true (default), all PVCs referenced by the target pod are deleted
- This simulates complete storage loss scenarios where data needs to be recreated or restored
- Use `deleteStorage: false` when you want to preserve data and test pod rescheduling without data loss

### Pod Targeting
- Pod replacement requires `level: pod` in the disruption specification
- Only one specific pod is targeted per disruption instance
- The target pod is identified by its IP address and must be in a running state

### Grace Period Handling
- If `gracePeriodSeconds` is specified, it overrides the pod's default grace period
- If `forceDelete` is true, the grace period is set to 0 regardless of other settings
- Graceful shutdown allows applications to clean up resources before termination

Setting `gracePeriodSeconds` to 0 or using `forceDelete` doesn't necessarily mean the pod will be terminated successfully. If it fails, its possible for the pod to continue running on the cluster indefinitely. All it does is removes the pod from the API so that a new pod with the same name can replace it, and gives it a small grace period before being force killed. You can read more about this in the [k8s docs](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination-forced)
