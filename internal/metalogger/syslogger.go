package metalogger

import (
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"log"
)

type Syslogger struct {
	Server     *syslog.Server
	Handler    *syslog.ChannelHandler
	Channel    chan format.LogParts
	Processors []Processor
	Writers    []Writer
}

type Processor interface {
	Process(parts format.LogParts) format.LogParts
}
type Writer interface {
	Write(parts format.LogParts)
}

func (s *Syslogger) Run() {
	s.Server.SetHandler(s.Handler)
	if err := s.Server.ListenUDP("0.0.0.0:514"); err != nil {
		log.Fatalln(err)
	}
	if err := s.Server.Boot(); err != nil {
		log.Fatalln(err)
	}
	for i := 0; i < 10; i++ {
		go func(channel syslog.LogPartsChannel) {
			for logParts := range channel {
				for _, p := range s.Processors {
					logParts = p.Process(logParts)
				}
				for _, w := range s.Writers {
					w.Write(logParts)
				}
			}
		}(s.Channel)
	}
	s.Server.Wait()
}
func New(processors []Processor, writers []Writer) Syslogger {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)

	return Syslogger{
		Server:     server,
		Handler:    handler,
		Channel:    channel,
		Processors: processors,
		Writers:    writers,
	}

}
