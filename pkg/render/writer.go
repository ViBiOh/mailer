package render

import (
	"bytes"
	"net/http"
)

// ResponseWriter implements http.ResponseWriter by storing content in memory
type ResponseWriter struct {
	status  int
	header  http.Header
	content *bytes.Buffer
}

// CreateWriter create ResponseWriter
func CreateWriter() *ResponseWriter {
	return &ResponseWriter{}
}

// Content return current contant buffer
func (rw *ResponseWriter) Content() *bytes.Buffer {
	return rw.content
}

// Status return current writer status
func (rw *ResponseWriter) Status() int {
	return rw.status
}

// SetStatus set current writer status
func (rw *ResponseWriter) SetStatus(status int) {
	rw.status = status
}

// Header cf. https://golang.org/pkg/net/http/#ResponseWriter
func (rw *ResponseWriter) Header() http.Header {
	if rw.header == nil {
		rw.header = http.Header{}
	}

	return rw.header
}

// Write cf. https://golang.org/pkg/net/http/#ResponseWriter
func (rw *ResponseWriter) Write(content []byte) (int, error) {
	if rw.content == nil {
		rw.content = bytes.NewBuffer(make([]byte, 0, 1024))
	}

	return rw.content.Write(content)
}

// WriteHeader cf. https://golang.org/pkg/net/http/#ResponseWriter
func (rw *ResponseWriter) WriteHeader(status int) {
	rw.status = status
}

// WriteResponse write memory content to real http ResponseWriter
func (rw *ResponseWriter) WriteResponse(w http.ResponseWriter) (int, error) {
	w.WriteHeader(rw.status)

	for key, values := range rw.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	return w.Write(rw.content.Bytes())
}
