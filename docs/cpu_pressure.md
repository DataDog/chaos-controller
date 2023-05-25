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

- It creates a dedicated process for each container seen in targeted pod:
  - We aligned CGroups Join implementation to join all cgroups on V1 and V2
  - When a container restarts, it imply processes in the same CGroups will be killed (SIGKILL)
  - To be able to reinject we hence use the standard injector process as an orchestrator that spins up one process per container and re-create them if needed
- Each newly created process (`/usr/local/bin/chaos-injector cpu-stress`) is hence responsible to perform the stress for a SPECIFIC container:
  - It parses the `cpuset.cpus` file (located in the target `cpuset` cgroup) to retrieve cores allocated to the targeted container
  - It calculates the percentage of stress to apply to all cores by taking user input `Count`
  - It then creates a dedicated goroutine per targeted core
    - Each goroutine is locked on the thread they are running on and their affinity is defined to a specific core
    - Each goroutine joins all CGroups
      - It's important to join cgroup (in particular `cpuset`) to both have the same number of allocated cores as the target as well as the same allocated cores so we ensure that the goroutines threads will be scheduled on the same cores as the target processes
      - We MUST join the targeted container CGroup and NOT creating another isolated CGroup with the same configuration as linux CPU is fair and ensure two separated processes will have their requested quotas even if one is trying to still all the CPUs
      - To guarantee the biggest throttling impact we hence need to be seen as part of the targetted process CGroup tree (we can't create a child CGroup either as it would prevent Kubernetes to manage the container as expected)
    - Each goroutine renices itself to the highest priority (`-20`) so the Linux scheduler will always give it the priority to consume CPU time over other running processes
    - Each goroutine starts an infinite loop to consume as much CPU as possible

> NB: stressing 100% of the allocated cpuset DOES NOT MEAN stressing 100% of all cores allocated if the defined CPU is below 1 in Kubernetes (e.g. `100m`)
> NB2: container being part of the same pods can have similar core associated, however we still need to stress each of them like if they were alone, linux CPU scheduler is the one that will throttling us appropriately

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

```sh
/ # cat /sys/fs/cgroup/cpuset/cpuset.cpus
0-5
/ # sysbench cpu run
sysbench 1.0.20-ac698b5ce3 (using bundled LuaJIT 2.1.0-beta2)

Running the test with following options:
Number of threads: 1
Initializing random number generator from current time


Prime numbers limit: 10000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:  5519.40

General statistics:
    total time:                          10.0004s
    total number of events:              55201

Latency (ms):
         min:                                    0.17
         avg:                                    0.18
         max:                                    0.64
         95th percentile:                        0.20
         sum:                                 9969.81

Threads fairness:
    events (avg/stddev):           55201.0000/0.00
    execution time (avg/stddev):   9.9698/0.00
```

Example `sysbench` with the 100% `examples/cpu_pressure.yaml` CPU pressure applied:

```sh
/ # cat /sys/fs/cgroup/cpuset/cpuset.cpus
0-5
/ # sysbench cpu run
sysbench 1.0.20-ac698b5ce3 (using bundled LuaJIT 2.1.0-beta2)

Running the test with following options:
Number of threads: 1
Initializing random number generator from current time


Prime numbers limit: 10000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:   196.13

General statistics:
    total time:                          10.0024s
    total number of events:              1962

Latency (ms):
         min:                                    0.17
         avg:                                    5.09
         max:                                   77.96
         95th percentile:                       59.99
         sum:                                 9979.32

Threads fairness:
    events (avg/stddev):           1962.0000/0.00
    execution time (avg/stddev):   9.9793/0.00
```

Example `sysbench` with the 1/6 core (16% per core) `examples/cpu_pressure.yaml` CPU pressure applied:

```sh
/ # cat /sys/fs/cgroup/cpuset/cpuset.cpus
0-5
/ # sysbench cpu run
sysbench 1.0.20-ac698b5ce3 (using bundled LuaJIT 2.1.0-beta2)

Running the test with following options:
Number of threads: 1
Initializing random number generator from current time


Prime numbers limit: 10000

Initializing worker threads...

Threads started!

CPU speed:
    events per second:  3308.87

General statistics:
    total time:                          10.0049s
    total number of events:              33107

Latency (ms):
         min:                                    0.17
         avg:                                    0.30
         max:                                   58.44
         95th percentile:                        0.21
         sum:                                 9974.76

Threads fairness:
    events (avg/stddev):           33107.0000/0.00
    execution time (avg/stddev):   9.9748/0.00
```

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

- Identify the injector process PIDs

```sh
# ps ax | grep chaos-injector
1113879 ?        Ssl    0:00 /usr/local/bin/chaos-injector cpu-pressure inject --metrics-sink noop --level pod --target-container-ids containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460,containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d --target-pod-ip 10.244.0.8 --chaos-namespace chaos-engineering --log-context-disruption-name dry-run --log-context-disruption-namespace chaos-demo --log-context-target-name demo-curl-547bb9c686-57484 --log-context-target-node-name lima --dry-run
1117684 pts/0    R+     0:00 grep chaos-injector
```

- Kill the injector processes

```sh
# kill 1113879
```

_You can SIGKILL the injector process if it is stuck but a standard kill is recommended._

- Ensure the injector process is gone

```sh
# ps ax | grep chaos-injector
1119071 pts/0    S+     0:00 grep chaos-injector
```
