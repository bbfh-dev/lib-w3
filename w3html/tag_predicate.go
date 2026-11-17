package w3html

import (
	"fmt"
)

type Predicate func(ctx RenderContext) bool

type predicateParser struct {
	Arguments []Argument
	length    uint
	i         uint
}

func ParsePredicate(arguments []Argument) (Predicate, error) {
	parser := predicateParser{
		Arguments: arguments,
		length:    uint(len(arguments)),
		i:         0,
	}
	return parser.Parse()
}

func (parser *predicateParser) Parse() (Predicate, error) {
	if len(parser.Arguments) == 0 {
		return nil, fmt.Errorf("empty predicate")
	}

	predicate, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	if parser.i != parser.length {
		return nil, fmt.Errorf(
			"unexpected argument at position %d: (%s) %s",
			parser.i+1,
			parser.Arguments[parser.i].Kind,
			parser.Arguments[parser.i],
		)
	}

	return predicate, nil
}

func (parser *predicateParser) parseExpression() (Predicate, error) {
	left, err := parser.parseComparison()
	if err != nil {
		return nil, err
	}

	for parser.i < parser.length &&
		parser.Arguments[parser.i].Kind == ArgLogicOperation {

		op := parser.Arguments[parser.i].AsLogicOperation()
		parser.i++

		right, err := parser.parseComparison()
		if err != nil {
			return nil, err
		}

		left = op.Compose(left, right)
	}

	return left, nil
}

func (parser *predicateParser) parseComparison() (Predicate, error) {
	if parser.i >= parser.length {
		return nil, fmt.Errorf("unexpected end of predicate")
	}

	left := parser.Arguments[parser.i]
	parser.i++

	if parser.i >= parser.length {
		return func(ctx RenderContext) bool {
			return ToBoolean(left.Compute(ctx))
		}, nil
	}

	opArg := parser.Arguments[parser.i]
	if opArg.Kind != ArgOperation {
		return func(ctx RenderContext) bool {
			return ToBoolean(left.Compute(ctx))
		}, nil
	}

	op := opArg.AsOperation()
	parser.i++

	if parser.i >= parser.length {
		return nil, fmt.Errorf(
			"expected right operand after operator %q at position %d",
			op.Token,
			parser.i,
		)
	}

	right := parser.Arguments[parser.i]
	parser.i++

	if op.Validator != nil {
		if err := op.Validator(left, right); err != nil {
			return nil, fmt.Errorf(
				"invalid operands for operator %q: %w",
				op,
				err,
			)
		}
	}

	if op.Comparator == nil {
		return nil, fmt.Errorf(
			"operator %q has no comparator",
			op.Token,
		)
	}

	return op.Comparator(left, right), nil
}
