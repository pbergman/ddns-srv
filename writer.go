package main

import (
	"context"
	"io"
	"sync"

	"github.com/pbergman/logger"
	"github.com/pbergman/provider"
)

type Writer struct {
	io.Writer
	mutex sync.Mutex
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.Writer.Write(p)
}

type ProviderDumper struct {
	provider []PluginProvider
	logger   *logger.Logger
	lock     WaitableLocker
}

func (d *ProviderDumper) WriteZones(ctx context.Context, stdout, stderr io.Writer, modules ...string) {
	WriteZones(ctx, d.provider, d.lock, stdout, stderr, modules...)
}

func (d *ProviderDumper) WriteRecords(ctx context.Context, stdout, stderr io.Writer, modules ...string) {
	WriteRecords(ctx, d.provider, d.lock, stdout, stderr, modules...)
}

func (d *ProviderDumper) WriteShort(ctx context.Context, stdout, stderr io.Writer, rtype, hostname string) {
	WriteShort(ctx, d.provider, d.lock, stdout, stderr, rtype, hostname)
}

func NewDumper(logger *logger.Logger, level provider.OutputLevel) (*ProviderDumper, error) {

	var _, providers, err = bootstrap(
		logger,
		inputOption("config", ""),
		level,
	)

	if err != nil {
		return nil, err
	}

	return &ProviderDumper{lock: NewSemaphore(5), provider: providers}, nil
}

func inSlice[T comparable](arr []T, item T) bool {

	for _, x := range arr {
		if x == item {
			return true
		}
	}

	return false
}
