# Generate Clientset

This guide outlines the detailed steps required to generate a clientset using the `client-gen` tool, a Kubernetes tool that automatically builds client libraries for working with Kubernetes API resources.

To begin, ensure that the Go source files, which contain API type definitions, include the necessary marker comments. These markers instruct `client-gen` to generate a client specifically tailored to these resources. Place the following markers at the top of the Custom Resource Definition (CRD) for which you need to generate a client:

```go
// +genclient
// +genclient:noStatus
// +genclient:onlyVerbs=create,get,list,delete,watch
```

- `// +genclient`: Activates client generation for the specified type.
- `// +genclient:noStatus`: Skips generating the `updateStatus` method even though a `status` field exists.
- `// +genclient:onlyVerbs=create,get,list,delete,watch`: Limits the client to only generate methods for these specified operations.

Including these comments ensures that the `client-gen` tool accurately generates a clientset according to your specific requirements for interaction with Kubernetes resources.

### Step-by-Step Guide

**1. Clone the `code-generator` Repository**

Start by cloning the `code-generator` repository from Kubernetes, which contains the `client-gen` tool.

```bash
git clone https://github.com/kubernetes/code-generator.git
```

**2. Checkout a Specific Version of the `code-generator`**

To ensure compatibility and stability, check out a specific version of the `code-generator`.

```bash
cd code-generator
git checkout v0.30.0-rc.1
```

**3. Build the `client-gen` Tool**

Compile the `client-gen` executable using Go. This step generates an executable in your current directory, allowing you to run `client-gen` directly.

```bash
go build -o client-gen ./cmd/client-gen
```

**4. Make `client-gen` Globally Available**

To use `client-gen` conveniently from any location on your system, move the executable to a directory included in your system's PATH, such as `/usr/local/bin`.

```bash
sudo mv client-gen /usr/local/bin/
```

**5. Navigate to the Project Repository**

Change directory to the location of `chaos-controller` repo where the API definitions are stored.

```bash
cd /path/to/chaos-controller/repo
```

**6. Generate the Clientset**

Run the `client-gen` command to generate the clientset. This command specifies where the input types are located (`--input-base`), which API versions to include (`--input`), the name of the generated clientset (`--clientset-name`), and the output location for the generated clientset files.

```bash
client-gen \
--input-base="github.com/DataDog/chaos-controller/api" \
--input="v1beta1" \
--clientset-name="v1beta1" \
--output-dir="$(pwd)/clientset" \
--output-pkg="github.com/DataDog/chaos-controller/clientset" \
--fake-clientset=false
```
