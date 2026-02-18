package main

import (
	"log"
	"net/http"
)

func main() {
	hub := newHub()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	log.Println("서버 시작: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
