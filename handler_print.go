package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pbergman/logger"
)

func NewPrintHandler(plugins []PluginProvider, logger *logger.Logger) Handler {
	return &PrintHandler{
		plugins: plugins,
		logger:  logger,
	}
}

type PrintHandler struct {
	plugins []PluginProvider
	logger  *logger.Logger
}

func (p *PrintHandler) Supports(_ *url.URL) bool {
	return true
}

func (p *PrintHandler) Handle(response http.ResponseWriter, request *http.Request) (result HandleResult) {

	response.Header().Set("content-type", "text/plain; charset=utf-8")

	var lock = NewSemaphore(5)
	var stdout = &Writer{
		Writer: response,
	}

	var stderr = io.MultiWriter(stdout, &Writer{
		Writer: p.logger.NewWriter(logger.Error),
	})

	if request.URL.Path == "/zones" {

		WriteZones(request.Context(), p.plugins, lock, stdout, stderr)

	} else if strings.HasPrefix(request.URL.Path, "/lookup/") {

		var parts = strings.SplitN(request.URL.Path[8:], "/", 2)

		var host string
		var rtype string

		if len(parts) == 2 {
			rtype, host = parts[0], parts[1]
		} else {
			rtype, host = "A", parts[0]
		}

		if x, err := url.QueryUnescape(host); err == nil {
			host = x
		}

		WriteShort(request.Context(), p.plugins, lock, stdout, stderr, strings.ToUpper(rtype), host)

	} else {

		WriteRecords(request.Context(), p.plugins, lock, stdout, stderr)
	}

	return StopPropagation
}
