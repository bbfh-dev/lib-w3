package w3errs

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	StatusCode   int
	InternalText string
	PublicText   string
}

func (err HttpError) Error() string {
	return fmt.Sprintf(
		"%s: %s",
		statusText(err.StatusCode),
		err.InternalText,
	)
}

func NewPublic(status_code int, text string) HttpError {
	return HttpError{
		StatusCode:   status_code,
		InternalText: text,
		PublicText:   text,
	}
}

func NewSimple(status_code int) HttpError {
	return NewPublic(status_code, statusText(status_code))
}

func NewInternal(status_code int, err error) HttpError {
	return HttpError{
		StatusCode:   status_code,
		InternalText: err.Error(),
		PublicText:   statusText(status_code),
	}
}

func statusText(status_code int) string {
	return fmt.Sprintf("%d %s", status_code, http.StatusText(status_code))
}
