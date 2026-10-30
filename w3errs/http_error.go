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

func NewPublic(statusCode int, text string) HttpError {
	return HttpError{
		StatusCode:   statusCode,
		InternalText: text,
		PublicText:   text,
	}
}

func NewSimple(statusCode int) HttpError {
	return NewPublic(statusCode, statusText(statusCode))
}

func NewInternal(statusCode int, err error) HttpError {
	return HttpError{
		StatusCode:   statusCode,
		InternalText: err.Error(),
		PublicText:   statusText(statusCode),
	}
}

func statusText(statusCode int) string {
	return fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
}
