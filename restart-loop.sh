#!/bin/bash

while true
do
    kubectl apply -f examples/minikube-scenarios/network-ingress-hosts.yaml
    sleep 15
    kubectl delete -f examples/minikube-scenarios/network-ingress-hosts.yaml
    sleep 10
done