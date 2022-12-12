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
	"regexp"
	"strconv"
	"sync"
	"time"
)

type TestProcessor struct{}

var re = regexp.MustCompile(`(?m)\d+`)
var founded = make(map[int]struct{})
var M sync.RWMutex
var T = time.Now()
var C = 0

func (t *TestProcessor) Process(parts format.LogParts) format.LogParts {
	for k, v := range parts {
		if k == "hostname" {
			parts["somethingExtra"] = "wow this is something else"
		}
		if k == "content" {
			for _, match := range re.FindAllString(v.(string), -1) {
				num, err := strconv.Atoi(match)
				if err != nil {
					fmt.Println(err)
				}
				M.Lock()
				founded[num] = struct{}{}
				prometheus.MessagesRecieved.Inc()
				C += 1
				M.Unlock()
				if num == 50000 {
					fmt.Printf("Finished Processing %v messages in %v\n", C, time.Since(T).String())
					spew.Dump(parts)
				}
			}
		}
	}
	return parts
}

type TestWriter struct{}

func (t *TestWriter) Write(parts format.LogParts) {
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
		metalogger.WithHealthCheckCadence(5*time.Second),
		metalogger.WithSocketSize(2560000),
		metalogger.WithFormat(&format.RFC3164{}),
		metalogger.WithPrometehusMetrics(8888),
	)
	t := time.NewTimer(time.Second * 10)
	go func() {
		for range t.C {
			fmt.Println("Checking for missing messages")
			for i := 0; i < 50000; i++ {
				M.RLock()
				if _, ok := founded[i+1]; !ok {
					fmt.Println("Missing message:", i)
				}
				M.RUnlock()

			}
			if C != 50000 {
				fmt.Println("Incorrect number of messages")
			}
		}
	}()
	s.Run()
}
