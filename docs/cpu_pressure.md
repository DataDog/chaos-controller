# CPU pressure

The `cpuPressure` field generates CPU load on the targeted pod.

## How it works

Containers achieve resource limitation (cpu, disk, memory) through cgroups. cgroups have the directory format `/sys/fs/cgroup/<kind>/<name>/`, and we can add a process to a cgroup by appending its `PID` to the `cgroups.procs` or the `tasks` files (depending on the use case). Docker containers get their own cgroup as illustrated by `PID 1873` below:

<p align="center"><kbd>
    <img src="img/cpu/cgroup_all.png" width=500 align="center" />
</kbd></p>

> :open_book: More information on how cgroups work [here](https://www.kernel.org/doc/Documentation/cgroup-v1/cgroups.txt).

The `/sys/fs/cgroup` directory of the host must be mounted in the injector pod at the `/mnt/cgroup` path for it to work.

<p align="center"><kbd>
    <img src="img/cpu/cgroup_injected.png" width=500 align="center" />
</kbd></p>

When the injector pod starts:

- It parses the `cpuset.cpus` file (located in the target `cpuset` cgroup) to retrieve cores allocated to the target processes.
- It calculates the number of cores to apply pressure, by taking user input `Count`.
- It then creates one goroutine per target core.
- Each goroutine is locked on the thread they are running on. By doing so, it forces the Go runtime scheduler to create one thread per locked goroutine.
- Each goroutine joins the target `cpu` and `cpuset` cgroups.
  - Joining the `cpuset` cgroup is important to both have the same number of allocated cores as the target as well as the same allocated cores so we ensure that the goroutines threads will be scheduled on the same cores as the target processes
- Each goroutine renices itself to the highest priority (`-20`) so the Linux scheduler will always give it the priority to consume CPU time over other running processes
- Each goroutine starts an infinite loop to consume as much CPU as possible

<p align="center"><kbd>
    <img src="img/cpu/cgroup_disrupted.png" width=500 align="center" />
</kbd></p>

## Manually Confirming CPU Pressure

In a CPU disruption, the injector pid is moved to the target CPU cgroup but the injector container keeps its own pid namespace. Commands like `top` or `htop` won't see the process running just because they can't see the pid, although it uses the cgroup CPU.

This can be confirmed using top linux command.

- If you run `top` from within the targeted pod, you won't see the CPU usage increasing nor the injector process running.
- If you run `top` from within the injector pod, you'll see the CPU usage increasing even though it is not consuming this container CPU resource.
- If you run `top` from the node where the pod is running, you will also see the injector process eating CPU.

This because those tools are mostly relying on processes they can see to display resource usage. This can also be confirmed with benchmarking tools such as `sysbench` running on the different containers.

Example `sysbench` without the CPU pressure applied:

```
root@demo-curl-8589cffd98-ccjqg:/# sysbench --test=cpu run
Running the test with following options:
Number of threads: 1
Initializing random number generator from current time


Prime numbers limit: 10000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:  1177.67

General statistics:
    total time:                          10.0004s
    total number of events:              11780

Latency (ms):
         min:                                  0.70
         avg:                                  0.85
         max:                                 18.00
         95th percentile:                      1.10
         sum:                               9975.80

Threads fairness:
    events (avg/stddev):           11780.0000/0.00
    execution time (avg/stddev):   9.9758/0.00
```

Example `sysbench` with the CPU pressure applied:

```
root@demo-curl-8589cffd98-ccjqg:/# sysbench --test=cpu run
Running the test with following options:
Number of threads: 1
Initializing random number generator from current time


Prime numbers limit: 10000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:   115.48

General statistics:
    total time:                          10.5973s
    total number of events:              1224

Latency (ms):
         min:                                  0.72
         avg:                                  8.65
         max:                                906.92
         95th percentile:                     74.46
         sum:                              10592.69

Threads fairness:
    events (avg/stddev):           1224.0000/0.00
    execution time (avg/stddev):   10.5927/0.00
```

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

- Identify the injector process PID

```
# ps ax | grep injector
   4376 ?        Ssl    7:42 /app/cmd/cainjector/cainjector --v=2 --leader-election-namespace=kube-system
1113879 ?        Ssl    0:00 /usr/local/bin/injector node-failure inject --metrics-sink noop --level pod --target-container-ids containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460,containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d --target-pod-ip 10.244.0.8 --chaos-namespace chaos-engineering --log-context-disruption-name dry-run --log-context-disruption-namespace chaos-demo --log-context-target-name demo-curl-547bb9c686-57484 --log-context-target-node-name lima --dry-run
1117684 pts/0    R+     0:00 grep injector
```

- Kill the injector process

```
# kill 1113879
```

_You can SIGKILL the injector process if it is stuck but a standard kill is recommended._

- Ensure the injector process is gone

```
# ps ax | grep injector
   4376 ?        Ssl    7:42 /app/cmd/cainjector/cainjector --v=2 --leader-election-namespace=kube-system
1119071 pts/0    S+     0:00 grep injector
```
