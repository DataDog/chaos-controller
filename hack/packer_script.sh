#!/bin/bash

# install minikube prerequisites
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update
sudo apt-get install -y build-essential gnupg2 p7zip-full git wget cpio python \
    unzip bc gcc-multilib automake libtool locales

# install go
wget https://golang.org/dl/go1.15.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.15.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version

# minikube
git clone https://github.com/kubernetes/minikube.git
cd minikube/
git checkout 9f1e48 # version 1.16.0

# configure kernel modules
cat <<EOT >> ./deploy/iso/minikube-iso/board/coreos/minikube/linux_defconfig
CONFIG_NET_SCH_PRIO=y
CONFIG_BLK_DEV_THROTTLING=y
EOT

# build ISO
IN_DOCKER=1 make out/minikube.iso

