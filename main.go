package main

import (
	"log"

	"lbryio/lbry-id/auth"
	"lbryio/lbry-id/env"
	"lbryio/lbry-id/server"
	"lbryio/lbry-id/store"
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
	srv := server.Init(&auth.Auth{}, &store, &env.Env{})
	srv.Serve()
}
