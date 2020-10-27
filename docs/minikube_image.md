# Custom Minikube Image

## Description

Some of the available disruptions (like `network_latency` and `network_limitation`) rely on the [`tc` command](http://man7.org/linux/man-pages/man8/tc.8.html) to function, which sets `qdiscs` on network devices in order to enforce extra network latency or bandwidth limitations.

These functions of `tc` rely on certain linux kernel modules being available, namely `SCH_TBF`, `SCH_NETEM`, and `SCH_PRIO`, the last of which isn't included in the base Minikube image that you'd get by installing and running Minikube normally. Thus, when testing changes locally, these disruptions wouldn't be able to operate and would crash with an opaque error under a standard Minikube cluster.

To work around this, we've created a custom Minikube ISO that contains not only `SCH_TBF` and `SCH_NETEM` (like the standard image) but also `SCH_PRIO` (which allows the latency disruption to work) and uploaded this image to S3. Then, passing in this image as the `--iso-url` parameter in `minikube start` solves the problem.

## Creating the Image

### [MacOS only] Prepare your environment

*Note: because of MacOS filesystem, you'll need to create a case-sensitive disk-image to build the ISO, otherwise it will fail.*

#### Create a case-sensitive disk-image

Copy [this script](https://gist.github.com/dixson3/8360571) in `/usr/local/bin/workspace`. Then, run the `workspace create` and `workspace attach` commands.

#### Create the Vagrant machine

Create the following `/Volumes/workspace/Vagrantfile` file:

```ruby
# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/bionic64"
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "16384"
    vb.cpus = 8
  end
end
```

Adjust the `vb.memory` and `vb.cpus` values to match your computer values so the virtual machine can use as much resources as possible for a faster build and run the `vagrant up` command to create and start it.

Once up and running, run the `vagrant ssh` command to SSH into the VM and run the `cd /vagrant` command to enter the mount point shared with your computer.

*Note: all commands below should be run from the /vagrant folder when using vagrant.*

### Install requirements

Follow the [prerequisites](https://minikube.sigs.k8s.io/docs/contrib/building/iso/#prerequisites) documentation to install needed components.

### Clone the repository using the right version

First, clone the Minikube Git repository **and checkout a released version**, not master:

```bash
git clone https://github.com/kubernetes/minikube.git
cd minikube/
git checkout 93af9c1 # version 1.9.2
```

### Configure additional kernel modules

Edit the `./deploy/iso/minikube-iso/board/coreos/minikube/linux_defconfig` file and add the follow lines at the end of the file:

```
CONFIG_NET_SCH_PRIO=y
CONFIG_BLK_DEV_THROTTLING=y
```

### Build the ISO

Finally, run the `IN_DOCKER=1 make out/minikube.iso` command to start the build. Please note that the image can be extremely long to build.

### [MacOS only] Cleanup

* Copy the `/Volumes/workspace/out/minikube.iso` file somewhere outside of the `workspace` volume.
* Run the `vagrant destroy` command to shutdown and delete the virtual machine.
* Run the `workspace detach` command to detach the `workspace` volume.

## Uploading the Image

> Note: Only Datadog employees currently have permissions to upload to the `public-chaos-controller` S3 bucket.

Finally, `aws s3 cp` the new ISO image into the `s3://public-chaos-controller/minikube/` directory, preferably using the `minikube-YYYY-MM-DD.iso` versioning format so as to avoid affecting existing, known-working images.

Then, update the `Makefile` of this repository to use the new image, once you've verified it works.
