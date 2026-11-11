package w3data

import (
	"reflect"
	"strings"
)

type Path []Segment

func (path Path) String() string {
	if len(path) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, segment := range path {
		builder.WriteRune('.')
		builder.WriteString(segment.String())
	}
	return builder.String()
}

type Segment struct {
	// Used for debugging only
	Word string
	// The index of the field/method in a struct
	Index uint
	// Whether the field is a method
	Call bool
	// Whether to iterate the value
	Iterate bool
}

func (segment Segment) String() string {
	var builder strings.Builder
	builder.WriteString(segment.Word)
	if segment.Call {
		builder.WriteString("()")
	}
	if segment.Iterate {
		builder.WriteString("[]")
	}
	return builder.String()
}

var noArgs = []reflect.Value{}

// Walk the [reflect.Value] using [Path].
// Panics on errors, the [Path] should be validated for this specific datatype beforehand.
func Walk(value reflect.Value, path Path) []reflect.Value {
	for i, segment := range path {
		if segment.Call {
			// Method
			method := value.Method(int(segment.Index))
			outputs := method.Call(noArgs)
			value = outputs[0]
			if segment.Iterate {
				goto iterate
			}
			continue
		}

		if !segment.Iterate {
			// Regular field
			value = value.Field(int(segment.Index))
			continue
		}

		value = value.Field(int(segment.Index))
	iterate:
		values := make([]reflect.Value, 0, value.Len())
		for j := range value.Len() {
			it := value.Index(j)
			values = append(values, Walk(it, path[i+1:])...)
		}
		return values
	}

	return []reflect.Value{value}
}
