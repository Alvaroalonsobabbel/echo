package main

import (
	"log"
	"net/http"

	"github.com/Alvaroalonsobabbel/echo/server"
	"github.com/Alvaroalonsobabbel/echo/store"
)

const port = ":3000"

func main() {
	store, err := store.New()
	if err != nil {
		log.Fatalf("unable to initialize storage: %v", err)
	}
	defer store.Close()
	if err := store.Seed(); err != nil {
		log.Fatalf("unable to seed the DB: %v", err)
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(port, server.New(store)))
}
