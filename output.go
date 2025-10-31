package main

import (
	"os"

	"github.com/pbergman/logger"
	"github.com/pbergman/provider"
)

func getOutput(debug bool, debugLevel int) (*logger.Logger, provider.OutputLevel) {

	var handler = make([]logger.HandlerInterface, 0)
	var level provider.OutputLevel = provider.OutputVerbose

	if debug {
		handler = append(handler, logger.NewWriterHandler(os.Stdout, logger.LogLevelDebug()^logger.LogLevelError(), false))
		handler = append(handler, logger.NewWriterHandler(os.Stderr, logger.LogLevelError(), false))
		level = provider.OutputLevel(debugLevel)
	} else {
		handler = append(handler, logger.NewThresholdHandler(
			logger.NewWriterHandler(os.Stdout, logger.LogLevelDebug(), false), 20, logger.Error, false,
		))
	}

	return logger.NewLogger("app", handler...), level
}
