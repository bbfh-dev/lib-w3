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

// HasName represents types whose name are known without reflection.
type HasName interface {
	Name() string
}

// GetName attemps to get a human-readable name of the datatype using reflection.
func GetName(handler any) string {
	if named, ok := handler.(HasName); ok {
		return named.Name()
	}

	r_value := reflect.ValueOf(handler)
	r_type := reflect.TypeOf(handler)
	switch r_type.Kind() {

	case reflect.Func:
		return GetFuncName(handler)

	case reflect.Struct:
		return GetStructName(r_type)

	case reflect.Pointer:
		r_type = r_type.Elem()
		if r_type.Kind() == reflect.Struct {
			return GetStructName(r_type)
		}
		fallthrough

	default:
		return r_value.String()
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

func GetStructName(r_type reflect.Type) string {
	name := r_type.Name()
	if len(name) == 0 {
		return r_type.String()
	}
	name = strings.ToUpper(name[:1]) + name[1:]

	_, after, ok := strings.CutLast(r_type.PkgPath(), "/")
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
