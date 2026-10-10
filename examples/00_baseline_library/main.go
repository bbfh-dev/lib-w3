package main

import (
	"net/http"
	"os"

	libw3 "codeberg.org/bbfh/lib-w3"
)

func main() {
	engine := libw3.NewBuilder().MustBuild()

	engine.GET("/:value", func(ctx *libw3.Context) {
		ctx.SendTextf(http.StatusOK, "Hello %s!", ctx.Params.ByName("value"))
	})

	err := engine.ListenAndServe(":8081")
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
}
