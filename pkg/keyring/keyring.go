package keyring

import (
	"log"

	kr "github.com/99designs/keyring"
)

var Store kr.Keyring

func init() {
	store, err := kr.Open(kr.Config{
		ServiceName: "postman-sync",
	})
	if err != nil {
		log.Fatal(err)
	}

	Store = store
}
