package main

import (
	"fmt"
	"log"

	"github.com/dsymonds/unifi"
)

func main() {
	api, err := unifi.NewAPI(unifi.FileAuthStore(unifi.DefaultAuthFile))
	if err != nil {
		log.Fatalf("unifi.NewClient: %v", err)
	}
	defer func() {
		if err := api.WriteConfig(); err != nil {
			log.Printf("api.WriteConfig: %v", err)
		}
	}()

	// TODO: make this automatic
	log.Printf("Logging in...")
	if err := api.Login(); err != nil {
		log.Fatalf("Logging in: %v", err)
	}

	log.Printf("Fetching clients...")
	clients, err := api.ListClients("default")
	if err != nil {
		log.Fatalf("Fetching clients: %v", err)
	}
	for _, client := range clients {
		fmt.Printf("%+v\n", client)
	}
}
