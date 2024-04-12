The Chaos Controller clientset is designed to facilitate easy and structured interactions with the Chaos Controller's Kubernetes resources. This section explains the high-level architecture of the clientset and the typed clients it includes.

The client has been generated using the client-gen ... (TODO: finish)

### The Clientset

A clientset is a collection of several typed clients, where each typed client is responsible for providing functions to interact with a specific group of Kubernetes API resources. The clientset designed for the Chaos Controller encapsulates all the client logic needed to interact with the various Chaos Controller resources, such as Disruptions and DisruptionCrons.

### Typed Clients Within the Clientset

The Chaos Controller clientset contains typed clients for each Chaos Controller resources, Disruption and DisruptionCrons which can be managed within a specified namespace. Each typed client provides a set of methods that correspond to the operations you can perform on its associated resources, such as `Create`, `Get`, `List`, `Delete`, and `Watch`.

### Usage Pattern

To interact with Chaos Controller resources, you typically follow these steps:

1. **Instantiate the `Clientset`** with your Kubernetes cluster configuration.
2. **Access the `ChaosClient`** via the `Chaos()` method of the `Clientset`.
3. **Use the `Disruptions` or `DisruptionCrons` method** of the `ChaosClient` to get an interface for the resource type you wish to manage.
4. Perform operations (Create, Get, List, Delete, Watch) on the resources using the methods provided by the respective interfaces.

### Example

Here's a simple example of listing all Disruption resources in a specific namespace:

```go
clientset, err := v1beta1.NewForConfig(config)
if err != nil {
    log.Fatal(err)
}

disruptions, err := clientset.Chaos().Disruptions("my-namespace").List(context.TODO(), metav1.ListOptions{})
if err != nil {
    log.Fatal(err)
}

for _, d := range disruptions.Items {
    fmt.Println("Found Disruption:", d.Name)
}
```
