package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/text/language"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tag := language.English
		fmt.Fprintf(w, "Lang: %s", tag)
	})
	http.ListenAndServe(":8080", r)
}
