package libw3

import "net/http"

// ResponseWriter is a custom wrapper that stores the [ResponseWriter.StatusCode].
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (writer *ResponseWriter) WriteHeader(code int) {
	if writer.StatusCode == 0 {
		writer.StatusCode = code
	}
	writer.ResponseWriter.WriteHeader(code)
}

func (writer *ResponseWriter) Write(p []byte) (int, error) {
	if writer.StatusCode == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(p)
}
