package w3html

type Predicate func(ctx RenderContext) bool

func ParsePredicate(arguments []Argument) Predicate {
	left := arguments[0]

	if len(arguments) == 1 {
		return func(ctx RenderContext) bool {
			return ToBoolean(left.Compute(ctx))
		}
	}

	middle := arguments[1]
	if middle.Kind != ArgOperation {
		return func(ctx RenderContext) bool {
			return ToBoolean(left.Compute(ctx))
		}
	}
	op := middle.AsOperation()

	right := arguments[2]
	return op.Comparator(left, right)
}
