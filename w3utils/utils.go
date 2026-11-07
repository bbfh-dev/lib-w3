package w3utils

import (
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

func IsStateModifyingMethod(method string) bool {
	switch method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:
		return true
	default:
		return false
	}
}

type FileLike interface {
	io.Reader
	Name() string
	Close() error
}

// HasName represents types whose name are known without reflection.
type HasName interface {
	Name() string
}

// GetName attemps to get a human-readable name of the datatype using reflection.
func GetName(handler any) string {
	if named, ok := handler.(HasName); ok {
		return named.Name()
	}

	rValue := reflect.ValueOf(handler)
	rType := reflect.TypeOf(handler)
	switch rType.Kind() {

	case reflect.Func:
		return GetFuncName(handler)

	case reflect.Struct:
		return GetStructName(rType)

	case reflect.Pointer:
		rType = rType.Elem()
		if rType.Kind() == reflect.Struct {
			return GetStructName(rType)
		}
		fallthrough

	default:
		return rValue.String()
	}
}

// GetFuncName returns the name of the function using reflection.
func GetFuncName(function any) string {
	name := runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name()
	_, after, ok := strings.CutLast(name, "/")
	if !ok {
		return name
	}
	return after
}

func GetStructName(rType reflect.Type) string {
	name := rType.Name()
	if len(name) == 0 {
		return rType.String()
	}
	name = strings.ToUpper(name[:1]) + name[1:]

	_, after, ok := strings.CutLast(rType.PkgPath(), "/")
	if !ok {
		return name
	}
	return after + "." + name
}

func SafeUint(value int) uint {
	if value < 0 {
		return 0
	}
	return uint(value)
}
