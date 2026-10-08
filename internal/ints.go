package internal

import "math"

func SafeUint8(n int) uint8 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(n)
}
