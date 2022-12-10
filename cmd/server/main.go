package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/metajar/metalogger/internal/metalogger"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

type TestProcessor struct{}

func (t *TestProcessor) Process(parts format.LogParts) format.LogParts {
	for k, v := range parts {
		if k == "hostname" {
			fmt.Println("looking up hostname ->", v)
			parts["somethingExtra"] = "wow this is something else"
		}
	}
	return parts
}

type TestWriter struct{}

func (t *TestWriter) Write(parts format.LogParts) {
	spew.Dump(parts)
}

func main() {
	s := metalogger.New([]metalogger.Processor{&TestProcessor{}}, []metalogger.Writer{&TestWriter{}})
	s.Run()
}
