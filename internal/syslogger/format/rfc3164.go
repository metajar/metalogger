package format

import (
	"bufio"
	"fmt"

	"github.com/metajar/metalogger/internal/syslogger/syslogparser/rfc3164"
)

type RFC3164 struct{}

func (f *RFC3164) GetParser(line []byte) LogParser {
	fmt.Printf("\n\n\n\n%v\n\n\n\n", string(line))
	return &parserWrapper{rfc3164.NewParser(line)}
}

func (f *RFC3164) GetSplitFunc() bufio.SplitFunc {
	return nil
}
