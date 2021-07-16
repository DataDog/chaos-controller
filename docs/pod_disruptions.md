# Pod failure

The `podFailure` field sends the SIGKILL/SIGINT signal to a pod's containers, thus killing/interrupting their main process.

The signal to be sent is controlled through the `podFailure.kill` field. If this is set to `true`, the SIGKILL signal will be sent. Otherwise the injector sends the SIGINT signal.

By default, all containers within a pod are targeted. However, you can target a predefined set of containers by setting the `containers` field.
