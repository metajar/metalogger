package healthchecks

import (
	"github.com/metajar/metalogger/internal/logger"
	"time"
)

// Self is a simple check that just identifies that the service is running.
// This is the most boring thing that we can possibly do.
type Self struct{}

func (s Self) Check() bool {
	return true
}

func (s Self) Success() {
	logger.SugarLogger.Infow("self check good!", "time", time.Now().String())
}

func (s Self) Failure() {
	return
}
