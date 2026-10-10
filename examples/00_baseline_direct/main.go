package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()

	router.GET("/:value", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello %s!", p.ByName("value"))
	})

	err := http.ListenAndServe(":8081", router)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
}
