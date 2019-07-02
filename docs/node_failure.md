# NodeFailureInjection ([example](../config/samples/chaos_v1beta1_nodefailureinjection.yaml))

The `NodeFailureInjection` triggers a kernel panic on the node. Because the node will reboot suddenly, the pods running on it (the injection pod included) wont' see their status updated for a while. This is why the injection pod can appear as `Running` or `Unknown` while it has currently finished the injection.
