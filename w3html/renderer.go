package w3html

import (
	"fmt"
	"reflect"
	"strings"
)

type RenderCommandEnum uint8

const methodOffset = 1 << 30

const (
	RenderBytes RenderCommandEnum = iota
)

type Renderer[T any] struct {
	Commands    []RenderCommandEnum
	CommandData []any
}

func Bake[T any](page ParsedPage) (*Renderer[T], error) {
	rType := reflect.TypeOf(*new(T))
	if rType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("unable to bake for %q of type %q", rType.Name(), rType.Kind())
	}

	renderer, err := bake[T](&page, rType)
	if err != nil {
		return nil, fmt.Errorf("baking for %q: %w", rType.Name(), err)
	}

	return renderer, nil
}

func bake[T any](page *ParsedPage, rType reflect.Type) (*Renderer[T], error) {
	for _, path := range page.MentionedPaths {
		_, err := resolve(rType, path, 0)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func resolve(rType reflect.Type, path StringPath, i int) ([]int, error) {
	segment := path[i]
	if before, ok := strings.CutSuffix(segment, "()"); ok {
		method, ok := rType.MethodByName(before)
		if !ok {
			return nil, fmt.Errorf(
				"document tries to call method %q, but %q is not present in type %q",
				path.Flatten(),
				segment,
				rType.Name(),
			)
		}
		if len(path) > i+1 {
			index, err := resolve(method.Type.Out(0), path, i+1)
			if err != nil {
				return nil, err
			}
			return append([]int{methodOffset + method.Index}, index...), nil
		}
		return []int{methodOffset + method.Index}, nil
	}

	if before, ok := strings.CutSuffix(segment, "[]"); ok {
		field, ok := rType.FieldByName(before)
		if !ok {
			return nil, fmt.Errorf(
				"document tries to access array field %q, but %q is not present in type %q",
				path.Flatten(),
				before,
				rType.Name(),
			)
		}
		if len(path) > i+1 {
			index, err := resolve(field.Type.Elem(), path, i+1)
			if err != nil {
				return nil, err
			}
			return append(field.Index, index...), nil
		}
		return field.Index, nil
	}

	field, ok := rType.FieldByName(segment)
	if !ok {
		return nil, fmt.Errorf(
			"document tries to access field %q, but %q is not present in type %q",
			path.Flatten(),
			segment,
			rType.Name(),
		)
	}
	if len(path) > i+1 {
		index, err := resolve(field.Type, path, i+1)
		if err != nil {
			return nil, err
		}
		return append(field.Index, index...), nil
	}
	return field.Index, nil
}

func read(rValue reflect.Value, path []int) any {
	segment := path[0]
	if segment >= methodOffset {
		method := rValue.Method(segment - methodOffset)
		out := method.Call([]reflect.Value{})[0]
		if len(path) > 1 {
			return read(out, path[1:])
		}
		return out
	}
	out := rValue.Field(segment)
	if len(path) > 1 {
		return read(out, path[1:])
	}
	return out
}
