package main

import (
	"io"
	"sync"
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
