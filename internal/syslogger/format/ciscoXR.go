package format

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/metajar/metalogger/internal/logger"
	"github.com/metajar/metalogger/internal/syslogger/syslogparser"
	"github.com/vjeantet/grok"
	"log"
	"time"
)

var GrokParser *grok.Grok

func init() {
	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	err := g.AddPatternsFromMap(CommonPatterns)
	if err != nil {
		log.Fatalln(err)
	}
	err = g.AddPattern("SYSLOG", `<%{INT:priority}>%{INT:sequence}:.*%{CISCOTIMESTAMP:log_date}: %{DATA:process}\[%{INT:pid}\]: %%{WORD:category}-%{WORD:group}-%{INT:severity}-%{WORD:mnemonic} : %{GREEDYDATA:message}`)
	if err != nil {
		log.Fatalln(err)
	}
	GrokParser = g
}

var CommonPatterns = map[string]string{
	"MONTH":          `\b(?:[Jj]an(?:uary|uar)?|[Ff]eb(?:ruary|ruar)?|[Mm](?:a|ä)?r(?:ch|z)?|[Aa]pr(?:il)?|[Mm]a(?:y|i)?|[Jj]un(?:e|i)?|[Jj]ul(?:y)?|[Aa]ug(?:ust)?|[Ss]ep(?:tember)?|[Oo](?:c|k)?t(?:ober)?|[Nn]ov(?:ember)?|[Dd]e(?:c|z)(?:ember)?)\b`,
	"MONTHDAY":       `(?:(?:0[1-9])|(?:[12][0-9])|(?:3[01])|[1-9])`,
	"YEAR":           `(?>\d\d){1,2}`,
	"TIME":           `(?!<[0-9])%{HOUR}:%{MINUTE}(?::%{SECOND})(?![0-9])`,
	"DATA":           `.*?`,
	"GREEDYDATA":     `.*`,
	"INT":            `(?:[+-]?(?:[0-9]+))`,
	"BASE10NUM":      `(?<![0-9.+-])(?>[+-]?(?:(?:[0-9]+(?:\.[0-9]+)?)|(?:\.[0-9]+)))`,
	"NUMBER":         `(?:%{BASE10NUM})`,
	"CISCOTIMESTAMP": `%{MONTH}.*[A-Z]`,
	"WORD":           `\b\w+\b`,
}

type CiscoXR struct {
	buff     []byte
	parsed   map[string]string
	Priority int    `json:"priority"`
	Sequence int    `json:"sequence"`
	Message  string `json:"message"`
	Severity int    `json:"severity"`
	Mnemonic string `json:"mnemonic"`
	LogDate  string `json:"log_date"`
	Process  string `json:"process"`
	Category string `json:"category"`
	Group    string `json:"group"`
}

func (f *CiscoXR) GetSplitFunc() bufio.SplitFunc {
	return nil
}

func (f *CiscoXR) Parse() error {
	m, err := GrokParser.Parse("%{SYSLOG}", string(f.buff))
	if err != nil {
		return err
	}
	f.parsed = m
	return nil
}

func (f *CiscoXR) Dump() syslogparser.LogParts {
	bs, err := json.Marshal(f.parsed)
	if err != nil {
		fmt.Println(err)
	}
	var m syslogparser.LogParts
	err = json.Unmarshal(bs, &m)
	if err != nil {
		logger.SugarLogger.Error(err)
	}
	return m
}

func (f *CiscoXR) Location(location *time.Location) {

}

func NewParser(line []byte) syslogparser.LogParser {
	return &CiscoXR{buff: line}
}

func (f *CiscoXR) GetParser(line []byte) LogParser {
	return &parserWrapper{NewParser(line)}
}

func (f *RFC3164) CiscoXR() bufio.SplitFunc {
	return nil
}
