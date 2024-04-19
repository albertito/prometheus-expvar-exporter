package main

import (
	"flag"
	"net/http"
)

var (
	addr = flag.String("addr", ":30081", "Address to listen on")
	path = flag.String("path", ".", "Path to serve")
)

func main() {
	flag.Parse()

	http.Handle("/", http.FileServer(http.Dir(*path)))
	http.ListenAndServe(*addr, nil)
}
