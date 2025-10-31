package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pbergman/logger"
)

type HandleResult bool

const (
	StopPropagation     HandleResult = false
	ContinuePropagation              = true
)

type Handler interface {
	Supports(url *url.URL) bool
	Handle(response http.ResponseWriter, request *http.Request) HandleResult
}

type ResponseWriter struct {
	http.ResponseWriter
	status int
}

func (r *ResponseWriter) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func NewServerHandler(config *ServerConfig, logger *logger.Logger, plugins map[string]ZoneAwareProvider) *ServerHandler {
	var trusted *IPPrefixList
	var handlers = []Handler{
		NewIconHandler(),
	}

	if nil != config {

		if nil != config.Users {
			handlers = append(handlers, NewAuthHandler(config.Users))
		}

		trusted = config.TrustedRemotes
	}

	return &ServerHandler{
		logger: logger,
		handlers: append(
			handlers,
			NewUpdateHandler(plugins, logger, trusted),
			NewPrintHandler(plugins, logger),
		),
	}
}

type ServerHandler struct {
	handlers []Handler
	logger   *logger.Logger
}

func (h *ServerHandler) log(start time.Time, response *ResponseWriter, request *http.Request) {
	var uri string

	if uri = request.RequestURI; uri == "" {
		uri = request.URL.RequestURI()
	}

	h.logger.Debug(fmt.Sprintf(
		"%s \"%s HTTP/%d.%d\" %d %s", request.Method, uri, request.ProtoMajor, request.ProtoMinor, response.status, time.Now().Sub(start).Round(time.Millisecond),
	))
}

func (h *ServerHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	var resp = &ResponseWriter{response, 200}
	var start = time.Now()

	defer h.log(start, resp, request)

	for i, c := 0, len(h.handlers); i < c; i++ {
		if h.handlers[i].Supports(request.URL) {
			if StopPropagation == h.handlers[i].Handle(resp, request) {
				return
			}
		}
	}
}
