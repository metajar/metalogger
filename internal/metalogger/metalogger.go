package metalogger

import (
	"github.com/metajar/metalogger/internal/logger"
	"github.com/metajar/metalogger/internal/metrics/prometheus"
	"github.com/metajar/metalogger/internal/syslogger"
	"github.com/metajar/metalogger/internal/syslogger/format"
	"time"
)

// MetaLogger is simply the main application that handles
// all the coordination in the system.
type MetaLogger struct {
	Server             *syslog.Server
	Handler            *syslog.ChannelHandler
	Channel            chan format.LogParts
	Processors         []Processor
	writers            []Writer
	HealthChecks       []HealthCheck
	healthCheckCadence time.Duration
	format             format.Format
	socketSize         int
	address            string
}

type Processor interface {
	Process(parts format.LogParts) format.LogParts
}
type Writer interface {
	Write(parts format.LogParts)
}

type HealthCheck interface {
	Check() bool
	Success()
	Failure()
}

func (s *MetaLogger) HealthCheckRoutine() {
	t := time.NewTicker(s.healthCheckCadence)
	for range t.C {
		for _, h := range s.HealthChecks {
			if h.Check() {
				h.Success()
			} else {
				h.Failure()
			}
		}
	}
}

// Run will take
func (s *MetaLogger) Run() {
	s.Server.SetHandler(s.Handler)
	if err := s.Server.ListenUDP(s.address); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	logger.SugarLogger.Infow("metalogger started up", "address", s.address)
	if err := s.Server.Boot(); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	go s.HealthCheckRoutine()
	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			logParts := logParts
			// Takes each message off the channel and throws it into its own goroutine.
			// This helps speed up the processing vs channel etc.
			go func() {
				for _, p := range s.Processors {
					logParts = p.Process(logParts)
				}
				for _, w := range s.writers {
					w.Write(logParts)
				}
			}()
		}
	}(s.Channel)
	s.Server.Wait()
}

type Option func(*MetaLogger)

func WithSocketSize(i int) Option {
	return func(s *MetaLogger) {
		s.socketSize = i
	}
}

func WithAddress(a string) Option {
	return func(s *MetaLogger) {
		s.address = a
	}
}

func WithHealthCheckCadence(t time.Duration) Option {
	return func(s *MetaLogger) {
		s.healthCheckCadence = t
	}
}

func WithProcessors(f []Processor) Option {
	return func(s *MetaLogger) {
		s.Processors = f
	}
}

func WithHealthChecks(f []HealthCheck) Option {
	return func(s *MetaLogger) {
		s.HealthChecks = f
	}
}
func WithWriters(f []Writer) Option {
	return func(s *MetaLogger) {
		s.writers = f
	}
}
func WithFormat(f format.Format) Option {
	return func(s *MetaLogger) {
		s.format = f
	}
}

func WithPrometehusMetrics(port int) Option {
	return func(s *MetaLogger) {
		prometheus.PromServer(port)
	}
}

// NewMetalogger will construct the new syslogger/metalogger that will be used
func NewMetalogger(opts ...Option) *MetaLogger {
	mlogger := &MetaLogger{}
	for _, opt := range opts {
		opt(mlogger)
	}
	channel := make(syslog.LogPartsChannel, 10000000)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	if mlogger.format == nil {
		logger.SugarLogger.Fatalln("a formatter will need to be set")
	}
	server.SetFormat(mlogger.format)
	server.SetSocketSize(mlogger.socketSize)
	mlogger.Server = server
	mlogger.Handler = handler
	mlogger.Channel = channel
	if mlogger.healthCheckCadence.Seconds() == 0 {
		mlogger.healthCheckCadence = time.Minute * 5
	}

	return mlogger
}
