package libw3

import (
	"slices"
	"sort"

	"codeberg.org/bbfh/lib-w3/w3utils"
)

type MiddlewareChain []Middleware

// ProperAppend should be used instead of using [append] directly. Read the function body to see why.
func (chain MiddlewareChain) ProperAppend(middleware Middleware) MiddlewareChain {
	if middleware, ok := middleware.(RequiresInit); ok {
		if keep := middleware.Init(); !keep {
			return chain
		}
	}
	return append(chain, middleware)
}

// Handlers returns [Handler] of contained [Middleware].
func (chain MiddlewareChain) Handlers() []Handler {
	handlers := make([]Handler, len(chain))
	for i, middleware := range chain {
		handlers[i] = middleware.Handle
	}
	return handlers
}

// Contains reports whether the [MiddlewareChain] contains [Middleware] by name.
func (chain MiddlewareChain) Contains(middleware Middleware) bool {
	return slices.Contains(chain.Names(), w3utils.GetName(middleware))
}

// Names returns human-readable names of contained [Middleware].
func (chain MiddlewareChain) Names() []string {
	names := make([]string, len(chain))
	for i, middleware := range chain {
		names[i] = w3utils.GetName(middleware)
	}
	return names
}

// Filtered returns the [MiddlewareChain] that's applicable for the specific handler method & path.
func (chain MiddlewareChain) Filtered(method, path string) MiddlewareChain {
	out := make(MiddlewareChain, 0, len(chain))
	for _, middleware := range chain {
		optional, ok := middleware.(Optional)
		if ok && !optional.ShouldInject(method, path) {
			continue
		}
		out = append(out, middleware)
	}
	return out
}

// Sort mutates the [MiddlewareChain] according to individual [Middleware]'s priority if provided.
func (chain MiddlewareChain) Sort() {
	sort.SliceStable(chain, func(i, j int) bool {
		a, a_ok := chain[i].(Ordered)
		b, b_ok := chain[j].(Ordered)

		var a_prio int8
		if a_ok {
			a_prio = a.Priority()
		}

		var b_prio int8
		if b_ok {
			b_prio = b.Priority()
		}

		return a_prio > b_prio
	})
}
