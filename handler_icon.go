package main

import (
	"crypto/sha1"
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	fileName = "favicon.ico"
)

var (
	//go:embed favicon.ico
	content embed.FS
)

func NewIconHandler() Handler {
	var buf []byte

	if bytes, err := content.ReadFile(fileName); err == nil {
		sha := sha1.New()
		sha.Write(bytes)
		buf = sha.Sum(nil)
	} else {
		buf, _ = time.Now().MarshalBinary()
	}

	return &IconHandler{etag: fmt.Sprintf("\"%x\"", buf)}
}

type IconHandler struct {
	etag string
}

func (u *IconHandler) Supports(url *url.URL) bool {
	return url.Path == "/"+fileName
}

func (u *IconHandler) Handle(response http.ResponseWriter, request *http.Request) HandleResult {

	if file, err := http.FS(content).Open(fileName); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	} else {

		if u.etag != "" {
			response.Header().Set("etag", u.etag)
		}

		http.ServeContent(response, request, fileName, time.Time{}, file)
	}

	return StopPropagation
}
