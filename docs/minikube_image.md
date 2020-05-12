# Custom Minikube Image

### Description

Some of the available disruptions (like `network_latency` and `network_limitation`) rely on the [`tc` command](http://man7.org/linux/man-pages/man8/tc.8.html) to function, which sets `qdiscs` on network devices in order to enforce extra network latency or bandwidth limitations.

These functions of `tc` rely on certain linux kernel modules being available, namely `SCH_TBF`, `SCH_NETEM`, and `SCH_PRIO`, the last of which isn't included in the base Minikube image that you'd get by installing and running Minikube normally. Thus, when testing changes locally, these disruptions wouldn't be able to operate and would crash with an opaque error under a standard Minikube cluster.

To work around this, we've created a custom Minikube ISO that contains not only `SCH_TBF` and `SCH_NETEM` (like the standard image) but also `SCH_PRIO` (which allows the latency disruption to work) and uploaded this image to S3. Then, passing in this image as the `--iso-url` parameter in `minikube start` solves the problem.

### Creating the Image

First, clone the Minikube Git repository **and checkout a released version, not master**:

```bash
git clone https://github.com/kubernetes/minikube.git && git checkout 93af9c1 # version 1.9.2
```

Then, modify this config file so that it includes the line `CONFIG_NET_SCH_PRIO=y` (the other necessary kernel modules are already included as of May 2020):

```bash
nano ./deploy/iso/minikube-iso/board/coreos/minikube/linux_defconfig
```

Finally, follow [the instructions](https://minikube.sigs.k8s.io/docs/contrib/building/iso/) to build a new ISO.

> Note: The image takes an _extremely long time_ to build in Docker on a normal laptop, upwards of 6 hours, and for me didn't work at all under OSX. It's highly recommended to stand up a fairly large (Ubuntu) EC2 instance, install build requirements, and run `make` with `IN_DOCKER=1` on that instance instead.

To provide an example of build time, I ran `make buildroot-image` and then `IN_DOCKER=1 make out/minikube.iso` on a `c5.9xlarge` Ubuntu EC2 instance (32 vCPU, 72GB RAM) for a modified Minikube 1.9.2, and the whole process took a little over an hour to produce the ISO.

### Uploading the Image
