// Command serve serves the web/ directory for the wasm build, with the
// application/wasm MIME type that WebAssembly.instantiateStreaming needs.
//
//	go run ./tools/serve [-addr :8080] [-dir web]
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dir := flag.String("dir", "web", "directory to serve")
	flag.Parse()
	mime.AddExtensionType(".wasm", "application/wasm")
	fs := http.FileServer(http.Dir(*dir))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	}))
	log.Printf("serving %s on http://localhost%s/", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
