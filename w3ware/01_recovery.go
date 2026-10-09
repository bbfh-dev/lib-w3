package w3ware

import (
	libw3 "codeberg.org/bbfh/lib-w3"
)

type recovery struct{}

var Recovery = &recovery{}

func (middleware *recovery) Handle(ctx *libw3.Context) {
}

func (middleware *recovery) Priority() int8 {
	return libw3.PrioRecovery
}
