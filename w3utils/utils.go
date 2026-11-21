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

func LooseEqual(a, b any) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if !av.IsValid() || !bv.IsValid() {
		return !av.IsValid() && !bv.IsValid()
	}

	if isNumeric(av.Kind()) && isNumeric(bv.Kind()) {
		return toFloat64(av) == toFloat64(bv)
	}

	if av.Type() == bv.Type() {
		if av.Type().Comparable() {
			return av.Interface() == bv.Interface()
		}

		return reflect.DeepEqual(a, b)
	}

	return false
}

func isNumeric(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func toFloat64(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return float64(v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint())

	case reflect.Float32, reflect.Float64:
		return v.Float()

	default:
		return 0
	}
}
