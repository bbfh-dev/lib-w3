package w3utils

// Holder to a slice element (slice index where the held data is stored).
// One should always be obtained from somewhere, rather than casted manually.
type Holder[T any] uint

// ————————————————————————————————

// Refer to [AllocateContextStoreHolder] and [Holder].
type ContextHolder[T any] Holder[T]

var contextStoreHolders uint = 0

// Allocates a new item to be stored in [libw3.Context]. Use the returned [ContextHolder] in [libw3.Context.Get] & [libw3.Context.Set] methods.
func AllocateContextStoreHolder[T any]() ContextHolder[T] {
	value := contextStoreHolders
	contextStoreHolders++
	return ContextHolder[T](value)
}

func ContextStoreHolders() uint {
	return contextStoreHolders
}

// ————————————————————————————————

type PageHolder[T any] Holder[T]
