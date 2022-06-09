package main

import (
	"log"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/server"
	"orblivion/lbry-id/store"
)

func storeInit() (s store.Store) {
	s = store.Store{}

	s.Init("sql.db")

	err := s.Migrate()
	if err != nil {
		log.Fatalf("DB setup failure: %+v", err)
	}

	return
}

func main() {
	store := storeInit()
	srv := server.Init(&auth.Auth{}, &store)
	srv.Serve()
}
