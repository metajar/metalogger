package format

import (
	"bufio"
	"fmt"

	"github.com/metajar/metalogger/internal/syslogger/syslogparser/rfc5424"
)

type RFC5424 struct{}

func (f *RFC5424) GetParser(line []byte) LogParser {
	fmt.Printf("\n\n\n\n%v\n\n\n\n", string(line))
	return &parserWrapper{rfc5424.NewParser(line)}
}

func (f *RFC5424) GetSplitFunc() bufio.SplitFunc {
	return nil
}
