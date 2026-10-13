package main

import (
	"fmt"
	"log"
	"net/http"
)

type Engine struct {
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		indexHandler(w, req)
	case "/hello":
		helloHandler(w, req)
	default:
		http.NotFound(w, req)
	}
}

func main() {
	log.Fatal(http.ListenAndServe(":9999", &Engine{}))
}

// handler echoes r.URL.Path
func indexHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
}

// handler echoes r.URL.Header
func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}
