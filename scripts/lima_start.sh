#!/usr/bin/env bash

# start the lima instance with the given image and get the kube config
limactl start --tty=false --name="${LIMA_INSTANCE}" - <"./${LIMA_CONFIG}.yaml"

# detect the current cgroup version running in the instance
CURRENT_CGROUP_FS=$(limactl shell "${LIMA_INSTANCE}" stat -fc %T /sys/fs/cgroup/)
if [[ ${CURRENT_CGROUP_FS} == "cgroup2fs" ]]; then
  CURRENT_CGROUPS="v2"
else
  CURRENT_CGROUPS="v1"
fi

echo "Detected cgroup version: ${CURRENT_CGROUPS}, desired: ${LIMA_CGROUPS}"

# reconfigure grub and restart only if the current version doesn't match the desired one
# we need to both call the reboot command and do a lima stop/start
# for the instance to be working (it rebinds things such as ssh to the host)
if [[ ${CURRENT_CGROUPS} != ${LIMA_CGROUPS} ]]; then
  if [[ ${LIMA_CGROUPS} == "v1" ]]; then
    echo "Reconfiguring lima instance with cgroups v1"
    limactl shell "${LIMA_INSTANCE}" sudo sed -i 's/GRUB_CMDLINE_LINUX="\(.*\)"/GRUB_CMDLINE_LINUX="\1 systemd.unified_cgroup_hierarchy=0"/' /etc/default/grub
  else
    echo "Reconfiguring lima instance with cgroups v2"
    limactl shell "${LIMA_INSTANCE}" sudo sed -i 's/GRUB_CMDLINE_LINUX="\(.*\)"/GRUB_CMDLINE_LINUX="\1 systemd.unified_cgroup_hierarchy=1"/' /etc/default/grub
  fi

  limactl shell "${LIMA_INSTANCE}" sudo update-grub
  limactl shell "${LIMA_INSTANCE}" sudo reboot
  echo "Waiting for instance to reboot, it might take a while"
  sleep 10
  limactl stop "${LIMA_INSTANCE}"
  limactl start "${LIMA_INSTANCE}"
else
  echo "Cgroup version already matches, no reconfiguration needed"
fi
