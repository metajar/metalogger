package metalogger

import (
	"github.com/metajar/metalogger/internal/logger"
	"github.com/metajar/metalogger/internal/syslogger"
	"github.com/metajar/metalogger/internal/syslogger/format"
	"time"
)

// Syslogger is simply the main application that handles
// all the coordination in the system.
type Syslogger struct {
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

func (s *Syslogger) HealthCheckRoutine() {
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

func (s *Syslogger) Run() {

	s.Server.SetHandler(s.Handler)
	if err := s.Server.ListenUDP("0.0.0.0:514"); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	logger.SugarLogger.Infow("metalogger started up", "port", "514")
	if err := s.Server.Boot(); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	go s.HealthCheckRoutine()
	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			logParts := logParts
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
func New(processors []Processor, writers []Writer) Syslogger {
	channel := make(syslog.LogPartsChannel, 10000000)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetSocketSize(2568576)

	return Syslogger{
		healthCheckCadence: time.Second * 30,
		Server:             server,
		Handler:            handler,
		Channel:            channel,
		Processors:         processors,
		writers:            writers,
	}

}

type SysloggerOption func(*Syslogger)

func WithSocketSize(i int) SysloggerOption {
	return func(s *Syslogger) {
		s.socketSize = i
	}
}

func WithAddress(a string) SysloggerOption {
	return func(s *Syslogger) {
		s.address = a
	}
}

func WithHealthCheckCadence(t time.Duration) SysloggerOption {
	return func(s *Syslogger) {
		s.healthCheckCadence = t
	}
}

func WithProcessors(f []Processor) SysloggerOption {
	return func(s *Syslogger) {
		s.Processors = f
	}
}

func WithHealthChecks(f []HealthCheck) SysloggerOption {
	return func(s *Syslogger) {
		s.HealthChecks = f
	}
}
func WithWriters(f []Writer) SysloggerOption {
	return func(s *Syslogger) {
		s.writers = f
	}
}
func WithFormat(f format.Format) SysloggerOption {
	return func(s *Syslogger) {
		s.format = f
	}
}

func NewMetalogger(opts ...SysloggerOption) *Syslogger {
	syslogger := &Syslogger{}
	for _, opt := range opts {
		opt(syslogger)
	}
	channel := make(syslog.LogPartsChannel, 10000000)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslogger.format)
	server.SetSocketSize(syslogger.socketSize)
	syslogger.Server = server
	syslogger.Handler = handler
	syslogger.Channel = channel

	return syslogger
}
