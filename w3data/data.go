package w3data

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
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

func (path Path) Flatten() string {
	str := path.String()
	if str == "" {
		return ""
	}
	return str[1:]
}

func (path Path) IsCompileTime() bool {
	return len(path) != 0 &&
		strings.ToLower(path[0].Word) == path[0].Word
}

type Segment struct {
	// Used for debugging only
	Word string
	// The index of the field/method in a struct
	Index uint
	// Whether the field is a method
	Call bool
	// Whether to iterate the value
	Iterate  bool
	dataType reflect.Type
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

func (segment Segment) DataType() reflect.Type {
	return segment.dataType
}

// ClearParserMetadata is a workaround for weird Go behavior during tests.
func (segment *Segment) ClearParserMetadata() {
	segment.dataType = nil
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

// ————————————————————————————————

func FindMethod(dataType reflect.Type, word string) (reflect.Method, error) {
	method, ok := dataType.MethodByName(word)
	if !ok {
		return method, fmt.Errorf("undefined method %q in %q", word, dataType.Name())
	}
	return method, nil
}

func FindField(dataType reflect.Type, word string) (reflect.StructField, error) {
	field, ok := dataType.FieldByName(word)
	if !ok {
		return field, fmt.Errorf("undefined field %q in %q", word, dataType.Name())
	}
	return field, nil
}

// ————————————————————————————————

func GetPath(dataType reflect.Type, segment Segment) (Path, error) {
	path, err := getPath(dataType, segment)
	if err != nil {
		var builder strings.Builder
		if segment.Call {
			builder.WriteString("()")
		}
		if segment.Iterate {
			builder.WriteString("[]")
		}
		return path, fmt.Errorf(
			"'(%s).%s%s': %w",
			dataType.String(),
			segment.Word,
			builder.String(),
			err,
		)
	}
	return path, nil
}

func getPath(dataType reflect.Type, segment Segment) (Path, error) {
	if strings.ToLower(string(segment.Word[0])) == string(segment.Word[0]) {
		return Path{segment}, nil
	}

	if segment.Call {
		method, err := FindMethod(dataType, segment.Word)
		if err != nil {
			return Path{}, err
		}
		if inputs := slices.Collect(method.Type.Ins()); len(inputs) == 0 {
			return Path{}, fmt.Errorf(
				"method must have no arguments, but found %d (%s)",
				len(inputs),
				inputs,
			)
		}
		outputs := slices.Collect(method.Type.Outs())
		if len(outputs) == 0 {
			return Path{}, errors.New("method must return at least one value")
		}
		if segment.Iterate {
			kind := outputs[0].Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return Path{}, fmt.Errorf(
					"method doesn't return a slice or array to iterate over (got %q)",
					kind,
				)
			}
		}
		return Path{Segment{
			Word:     segment.Word,
			Index:    uint(method.Index),
			Call:     true,
			Iterate:  segment.Iterate,
			dataType: method.Type,
		}}, nil
	}

	if dataType.Kind() != reflect.Struct {
		return Path{}, fmt.Errorf(
			"can't access field %q of a non-struct type %q",
			segment.Word,
			dataType.String(),
		)
	}

	field, err := FindField(dataType, segment.Word)
	if err != nil {
		return Path{}, err
	}
	if segment.Iterate {
		kind := field.Type.Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return Path{}, fmt.Errorf(
				"field is not a slice or array to iterate over (got %q)",
				kind,
			)
		}
	}
	path := Path{}
	last := len(field.Index) - 1
	for _, i := range field.Index {
		it := dataType.Field(i)
		path = append(path, Segment{
			Word:     it.Name,
			Index:    uint(i),
			Call:     false,
			Iterate:  segment.Iterate && i == last,
			dataType: it.Type,
		})
		dataType = it.Type
	}
	return path, nil
}
