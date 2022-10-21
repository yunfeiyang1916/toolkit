package main

import (
	log "github.com/yunfeiyang1916/toolkit/logging"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	http.HandleFunc("/hehe", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("呵呵"))
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
