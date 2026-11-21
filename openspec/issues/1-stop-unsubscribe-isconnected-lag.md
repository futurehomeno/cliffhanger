# Stop gate on IsConnected lags the paho connection state

`doStop` skips the unsubscriptions when `fimpgo.MqttTransport.IsConnected()` is false. fimpgo clears
that flag only in its connection-lost handler, which paho calls after the drop is detected. A probe
showed `IsConnected()` still true right after the broker closed the connection, while `Unsubscribe`
already failed. A stop that lands inside that window can still return an unsubscribe error.

Open question: is the window acceptable, or should `doStop` also ignore paho's not-connected errors
(no exported sentinel for `not currently connected and ResumeSubs not set`)?
