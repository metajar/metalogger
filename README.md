# Metalogger

Summary:

Metalogger is a simple and fast syslog server that can be used within a network to write to multiple destinations and
run a large set of processors on
This allows for a quick and simple way to add a syslog server in any network.

## Processors

Example LogParts:

```
(format.LogParts) (len=10) {
 (string) (len=7) "content": (string) (len=29) "Test UDP syslog message 50000",
 (string) (len=8) "priority": (int) 14,
 (string) (len=14) "somethingExtra": (string) (len=26) "wow this is something else",
 (string) (len=8) "hostname": (string) "",
 (string) (len=3) "tag": (string) "",
 (string) (len=8) "severity": (int) 6,
 (string) (len=6) "client": (string) (len=20) "172.31.255.119:47690",
 (string) (len=8) "tls_peer": (string) "",
 (string) (len=9) "timestamp": (time.Time) 2022-12-12 00:19:57 +0000 UTC,
 (string) (len=8) "facility": (int) 1
}
```

Processors will simply take in the syslog message and perform processing on it. This can be anything
from reverse DNS lookup, to device lookup, interface matching and whatver your use case may be. Adding
Processors to the metalogger must satisfy the processor interface:

```go
type Processor interface {
Process(parts format.LogParts) format.LogParts
}

```

They then can be added to the metalogger by passing them in at instantiation.

```go
s := metalogger.NewMetalogger(
metalogger.WithProcessors([]metalogger.Processor{&TestProcessor{}}),
}

```

# Writers

Writers can be added to the system to handle what to do with the messages once
they are completed. The writer will just need to follow the writer interface:

```go
type Writer interface {
Write(parts format.LogParts)
}
```

Writer will write the messages to a destination of your choice. The format of
LogParts is a `Map[string]interface{}` so they technically can be marshalled in any
format. It is worthwhile that they should however be batched to the destination.

# HealthChecks

HealthChecks allows the system a way to perform some type of health check and perform a
success or failure action. The healthcheck must satisfy the interface:

```go 
type HealthCheck interface {
	Check() bool
	Success()
	Failure()
}
```

Healthchecks are useful to signal if a destination writer can't be reached or any other checks.
One interesting case is the fact that if you have the BGP plugin installed you can stop
announcing the anycast IP address of the syslog server if problems arise preventing hosts
sending syslog traffic to it. 