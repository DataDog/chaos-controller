# Container failure

The `containerFailure` field sends the SIGTERM/SIGKILL signal to a pod's containers, thus graceful/non-gracefully terminating their main process.

The signal to be sent is controlled through the `containerFailure.forced` field. By default, this is set to `false` which will send the `SIGTERM` signal. If this is enabled the `SIGKILL` signal will be sent.

By default, all containers within a pod will be targeted. However, you can target a predefined set of containers by setting the `containers` field.
