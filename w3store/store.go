package w3store

// Holder to a slice element (slice index where the held data is stored).
// One should always be obtained from somewhere, rather than set manually.
//
// Refer to [AllocateContextStoreHolder].
type Holder[T any] uint

var contextStoreHolders uint = 0

// Allocates a new item to be stored in [libw3.Context]. Use the returned [Holder] in [libw3.Context.Get] & [libw3.Context.Set] methods.
func AllocateContextStoreHolder[T any]() Holder[T] {
	value := contextStoreHolders
	contextStoreHolders++
	return Holder[T](value)
}

func ContextStoreHolders() uint {
	return contextStoreHolders
}
