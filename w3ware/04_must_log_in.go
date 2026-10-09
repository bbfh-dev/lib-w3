package w3ware

import libw3 "codeberg.org/bbfh/lib-w3"

type mustLogIn struct{}

var MustLogIn = &mustLogIn{}

func (middleware *mustLogIn) Handle(ctx *libw3.Context) {
}

func (middleware *mustLogIn) Priority() int8 {
	return libw3.PrioValidate
}
