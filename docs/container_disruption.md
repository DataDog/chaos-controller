# Container failure

The `containerFailure` field sends the SIGTERM/SIGKILL signal to a pod's containers, thus graceful/non-gracefully terminating their main process.

The signal to be sent is controlled through the `containerFailure.forced` field. By default, this is set to `false` which will send the `SIGTERM` signal. If this is enabled the `SIGKILL` signal will be sent.

If those containers are restarted during the duration of the disruption, we will find their new PIDs and send the signal again.
This can achieve an effect of "continually restarting" containers within a pod.

By default, all containers within a pod will be targeted. However, you can target a predefined set of containers by setting the `containers` field.
