# Clientset

The Chaos Controller clientset is a structured interface that simplifies interactions with the Chaos Controller's Kubernetes resources. It featured dedicated typed clients, which enable straightforward operations on chaos resources such as Disruptions and DisruptionCrons. This document provides further details on the clientset's architecture, its typed clients, and their usage.

## Clientset Generation

The client is created with `client-gen`, a Kubernetes tool that automatically builds client libraries for working with Kubernetes API resources. For more detailed information about client-gen, including its flags and other usage details, you can visit the [official Kubernetes documentation on generating clientsets](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/generating-clientset.md).

## Clientset Architecture

A clientset groups together multiple typed clients, each tailored to interact with a distinct set of Kubernetes API resources. For the Chaos Controller, the clientset is designed to encapsulate all the necessary logic for managing custom resources, specifically Disruptions and DisruptionCrons.

These typed clients operate within a specified namespace and offer methods that mirror Kubernetes operations. These operations include `Create`, `Get`, `List`, `Delete`, and `Watch`.

## Usage

Working with Chaos Controller resources via the clientset typically involves the following steps:

1. **Create the `Clientset`**: Start by initializing a `Clientset` instance with your Kubernetes cluster configuration. This step establishes the connection to your cluster, allowing subsequent operations on resources.

2. **Access `ChaosClient`**: Call the `Chaos()` method on the `Clientset` to obtain a `ChaosClient` instance. This client is designed to handle Chaos Controller specific resources.

3. **Manage Resources**: Depending on your needs, use `Disruptions(ns)` or `DisruptionCrons(ns)` on the `ChaosClient` to get an interface for managing either Disruptions or DisruptionCrons, where `ns` is the target namespace. These functions grant access to interfaces for handling Disruptions or DisruptionCrons, respectively, providing you with the necessary tools for resource manipulation.

4. **Perform Operations**: Once you have the right interface, you can create, get, list, delete, or watch resources using using the methods it provides.

### Example

The following example demonstrates how to use the clientset to list Disruptions and DisruptionCrons:

```go
// Create the clientset using the Kubernetes configuration
clientset, err := v1beta1.NewForConfig(config)
if err != nil {
    log.Fatalf("Failed to create clientset: %v", err)
}

// Use the clientset to list all Disruptions in "my-namespace"
disruptions, err := clientset.Chaos().Disruptions("my-namespace").List(context.TODO(), metav1.ListOptions{})
if err != nil {
    log.Fatalf("Failed to list disruptions: %v", err)
}

// Use the clientset to list all DisruptionCrons in "my-namespace"
disruptionCrons, err := clientset.Chaos().DisruptionCrons("my-namespace").List(context.TODO(), metav1.ListOptions{})
if err != nil {
    log.Fatalf("Failed to list disruption crons: %v", err)
}
```
