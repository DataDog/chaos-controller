# CPU pressure

The `cpuPressure` fields generates CPU load on the targeted pod. The injector pod starts and joins the targeted pod container CPU cgroup. It then starts as many goroutines as available CPU (or specified limit) and starts an infinite loop.

The injector increases its own priority to the maximum value (`-20`) to ensure to take as much CPU time as possible. It is done before goroutines start and on the whole process group, so every threads created the injector also have the maximum priority.

## Joining the container CPU cgroup

To join the CPU cgroup, the injector will:
* retrieve the targeted pod container cgroup path from the container data
* write its own [process group ID](https://linux.die.net/man/3/getpgid) to the `cgroup.procs` file

The `/sys/fs/cgroup` directory of the host must be mounted in the injector pod at the `/mnt/cgroup` path for it to work.

More information on how cgroups work [here](https://www.kernel.org/doc/Documentation/cgroup-v1/cgroups.txt).
