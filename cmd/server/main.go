package main

import (
	"fmt"
	"github.com/metajar/metalogger/internal/metalogger"
	"github.com/metajar/metalogger/internal/syslogger/format"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type TestProcessor struct{}

var re = regexp.MustCompile(`(?m)\d+`)
var founded = make(map[int]struct{})
var M sync.Mutex
var T = time.Now()

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
				M.Unlock()
				if num == 50000 {
					fmt.Println("Finished Processing: ", time.Since(T).String())
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
	s := metalogger.New([]metalogger.Processor{&TestProcessor{}}, []metalogger.Writer{&TestWriter{}})
	t := time.NewTimer(time.Second * 60)
	go func() {
		for range t.C {
			fmt.Println("Checking for missing messages")
			for i := 0; i < 50000; i++ {
				if _, ok := founded[i+1]; !ok {
					fmt.Println("Missing message:", i)
				}

			}
		}
	}()
	s.Run()
}
