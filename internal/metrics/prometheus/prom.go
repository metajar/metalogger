package prometheus

import (
	"fmt"
	"github.com/metajar/metalogger/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	MessagesRecieved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metalogger_messages_recieved",
		Help: "The total number of processed messages",
	})
)

func PromServer(port int) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
		if err != nil {
			logger.SugarLogger.Error("could not start the metrics server")
		}
	}()

}
