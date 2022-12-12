package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/metajar/metalogger/internal/healthchecks"
	"github.com/metajar/metalogger/internal/healthchecks/gobgp"
	"github.com/metajar/metalogger/internal/metalogger"
	"github.com/metajar/metalogger/internal/metrics/prometheus"
	"github.com/metajar/metalogger/internal/syslogger/format"
	apipb "github.com/osrg/gobgp/v3/api"
	"time"
)

type TestProcessor struct{}

func (t *TestProcessor) Process(parts format.LogParts) format.LogParts {
	for k, v := range parts {
		if k == "hostname" {
			parts["somethingExtra"] = fmt.Sprintf("this is something that would be processed for %v", v)
		}
		prometheus.MessagesRecieved.Inc()
	}
	return parts
}

type TestWriter struct{}

func (t *TestWriter) Write(parts format.LogParts) {
	spew.Dump(parts)
}

func main() {
	// We can setup a bgp healthcheck. The amin thing is as long
	// as the application is running a BGP session will be setup and
	// the AnnouncePrefix will be sent to BGP peer with the next hop
	// set to itself. This is useful for setting up an anycast address
	// with multiple syslog servers.
	bgpHealth := gobgp.New(
		gobgp.RouterID("172.31.255.119"),
		gobgp.RouterASN(64512),
		gobgp.NeighborAddress("192.168.88.2"),
		gobgp.NeighborASN(65001),
		gobgp.AnnouncePrefix(&apipb.IPAddressPrefix{
			PrefixLen: 32,
			Prefix:    "10.10.10.10",
		}),
		gobgp.WithEbgpMulti(255),
	)

	// Setup the main metalogger.
	s := metalogger.NewMetalogger(
		metalogger.WithProcessors([]metalogger.Processor{&TestProcessor{}}),
		metalogger.WithWriters([]metalogger.Writer{&TestWriter{}}),
		metalogger.WithAddress("0.0.0.0:514"),
		metalogger.WithHealthChecks([]metalogger.HealthCheck{healthchecks.Self{}, bgpHealth}),
		metalogger.WithHealthCheckCadence(10*time.Second),
		metalogger.WithSocketSize(2560000),
		metalogger.WithFormat(&format.Automatic{}),
		metalogger.WithPrometehusMetrics(8888),
	)
	s.Run()
}
