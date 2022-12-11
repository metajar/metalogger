package logger

import (
	"go.uber.org/zap"
)

var SugarLogger *zap.SugaredLogger

func init() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()
	SugarLogger = sugar

}
