#!/bin/bash

readonly pod=$1
if [[ -z "$pod" ]]; then
  echo "missing parameter: $0 POD_NAME"
  exit 1
fi

# get container id
id=$(kubectl get -ojson pod $pod | jq -r '.status.containerStatuses[0].containerID' | sed 's#containerd://##')

# get pid
inspect=$(minikube ssh -- "sudo crictl inspect $id")
pid=$(echo "$inspect" | jq -r .info.pid)

# list iptables
minikube ssh -- "sudo nsenter --net=/proc/$pid/ns/net iptables -L -nv"
