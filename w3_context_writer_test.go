package libw3_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libw3 "codeberg.org/bbfh/lib-w3"
)

func TestStatusWriter(t *testing.T) {
	tests := []struct {
		name   string
		handle func(http.ResponseWriter)
		want   int
	}{
		{
			name: "explicit status",
			handle: func(writer http.ResponseWriter) {
				writer.WriteHeader(http.StatusNotFound)
			},
			want: http.StatusNotFound,
		},
		{
			name: "implicit OK on write",
			handle: func(writer http.ResponseWriter) {
				_, _ = writer.Write([]byte("hello"))
			},
			want: http.StatusOK,
		},
		{
			name: "first status wins",
			handle: func(writer http.ResponseWriter) {
				writer.WriteHeader(http.StatusCreated)
				writer.WriteHeader(http.StatusNotFound)
			},
			want: http.StatusCreated,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writer := &libw3.ResponseWriter{ResponseWriter: recorder}

			test.handle(writer)

			if got := writer.StatusCode; got != test.want {
				t.Errorf("wrapper status = %d, want %d", got, test.want)
			}
			if got := recorder.Result().StatusCode; got != test.want {
				t.Errorf("response status = %d, want %d", got, test.want)
			}
		})
	}
}
