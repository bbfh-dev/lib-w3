package w3utils

import (
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

func GetName(handler any) string {
	r_value := reflect.ValueOf(handler)
	switch reflect.TypeOf(handler).Kind() {

	case reflect.Func:
		name := runtime.FuncForPC(r_value.Pointer()).Name()
		_, after, ok := strings.CutLast(name, "/")
		if !ok {
			return name
		}
		return after

	default:
		return r_value.String()
	}
}
