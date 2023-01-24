# Applying gRPC disruption to your own Application

:warning: For a step-by-step tutorial to practice verifying a gRPC disruption on our demo application, follow [these instructions](demo_instructions.md).

### (0) Import disruption packages

Say you currently have an application which creates a gRPC server and registers your service and then listens on `serverAddr`:

```
lis, err := net.Listen("tcp", serverAddr)
if err != nil {
	log.Fatalf("failed to listen: %s\n", err)
}

dogfoodServer := grpc.NewServer()

df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})

if err := dogfoodServer.Serve(lis); err != nil {
	log.Fatalf("failed to serve: %v", err)
}
```

`chaos-controller` will inject a disruption to your application by communicating with an interceptor which executes code every time a query reaches the gRPC server. It communicates with your server through a new service dedicated to chaos called `disruptionlistener` which you need to register upon server instantiation along with the interceptor code. Both code snippets already exist in this repository and simply need to be imported:

```
import (
    <your other imports>

	disruption_service "github.com/DataDog/chaos-controller/grpc"
	dl_pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
)
```

### (2) Create a chaos feature flag

Next, create a feature flag for your interceptor. The gRPC disruption is safe, but putting the code for the interceptor behind a feature flag allows you to quickly redeploy your application without the `ChaosServerInterceptor`.

```
var dogfoodServer *grpc.Server

if <CHAOS_FEATURE_FLAG> == true {
    dogfoodServer = grpc.NewServer()

    df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})

} else {
    dogfoodServer = grpc.NewServer()

    df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})
}

if err := dogfoodServer.Serve(lis); err != nil {
    log.Fatalf("failed to serve: %v", err)
}
```

### (3) Integrate interceptor to server instantiation code behind chaos feature flag

Finally:
* create a new `disruptionListener`
* add the `ChaosServerInterceptor` when instantiating your gRPC Server
* register the `disruptionListener` to your gRPC Server

```
var dogfoodServer *grpc.Server

if <CHAOS_FEATURE_FLAG> == true {
	fmt.Println("CHAOS ENABLED")

    disruptionListener := disruption_service.NewDisruptionListener(logger)

	dogfoodServer = grpc.NewServer(
		grpc.UnaryInterceptor(disruptionListener.ChaosServerInterceptor),
	)

    df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})
    dl_pb.RegisterDisruptionListenerServer(dogfoodServer, disruptionListener)
} else {
    dogfoodServer = grpc.NewServer()

    df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})
}

if err := dogfoodServer.Serve(lis); err != nil {
    log.Fatalf("failed to serve: %v", err)
}
```

You can pass in a logger which you may have instantiated with: `	loggerConfig := zap.NewProductionConfig(); logger, err := loggerConfig.Build()` ([chaos-controller/log](../../log) contains sample logger code).

### (4) Apply gRPC disruption

Define a disruption using [examples/grpc.yaml](../../examples/grpc.yaml) as an example and save it locally. make sure you have the right `namespace` and `selector`

Run `kubectl apply -f <disruption>.yaml` to apply the disruption.
Run `kubectl delete -f <disruption>.yaml` to remove the disruption.
