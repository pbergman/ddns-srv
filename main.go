package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	flag.Parse()

	var logger, level = getOutput(
		inputOption("debug", false),
		inputOption("provider-debug-level", 2),
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Error(r)
			os.Exit(1)
		}
	}()

	var ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	var config, providers, err = bootstrap(
		logger,
		inputOption("config", ""),
		level,
	)

	if err != nil {
		panic(err)
	}

	var srv = NewServer(ctx, config, logger, providers)

	go func() {
		logger.Debug(fmt.Sprintf("listening on %s", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && false == errors.Is(err, http.ErrServerClosed) {
			logger.Error(err.Error())
		}
	}()

	<-ctx.Done()

	stop()

	shutdown(srv)
}

func shutdown(srv *http.Server) {
	var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
}
