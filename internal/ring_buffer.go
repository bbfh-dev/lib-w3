package internal

const ringBufferCapacity = 1024

// A ring buffer with fixed capacity of 1024 elements.
// Refer to [NewRingBuffer].
type RingBuffer[T any] struct {
	array   [ringBufferCapacity]T
	current uint
	next    uint
	size    uint
}

func NewRingBuffer[T any]() *RingBuffer[T] {
	return &RingBuffer[T]{
		array: [ringBufferCapacity]T{},
		next:  0,
		size:  0,
	}
}

func (buffer *RingBuffer[T]) Push(value T) {
	buffer.current = buffer.next
	buffer.next = (buffer.next + 1) % ringBufferCapacity
	buffer.array[buffer.current] = value
	buffer.size = min(buffer.size+1, ringBufferCapacity)
}

func (buffer *RingBuffer[T]) Dump() []T {
	result := make([]T, 0, buffer.size)
	var start uint = 0
	if buffer.size == ringBufferCapacity {
		start = buffer.next
	}

	for i := uint(0); i < buffer.size; i++ {
		index := (start + i) % ringBufferCapacity
		result = append(result, buffer.array[index])
	}

	return result
}
