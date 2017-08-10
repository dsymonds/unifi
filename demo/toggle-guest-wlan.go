package main

import (
	"log"
	"os"

	"github.com/dsymonds/unifi"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s [on|off]", os.Args[0])
	}
	var enable bool
	switch os.Args[1] {
	case "on", "true", "yes":
		enable = true
	case "off", "false", "no":
		enable = false
	default:
		log.Fatalf("usage: %s [on|off]", os.Args[0])
	}

	api, err := unifi.NewAPI(unifi.FileAuthStore(unifi.DefaultAuthFile))
	if err != nil {
		log.Fatalf("unifi.NewClient: %v", err)
	}
	defer func() {
		if err := api.WriteConfig(); err != nil {
			log.Printf("api.WriteConfig: %v", err)
		}
	}()

	const site = "default"

	log.Printf("Fetching wireless networks...")
	wlans, err := api.ListWirelessNetworks(site)
	if err != nil {
		log.Fatalf("Fetching wireless networks: %v", err)
	}
	for _, wlan := range wlans {
		if wlan.Guest {
			err := api.EnableWirelessNetwork(site, wlan.ID, enable)
			if err != nil {
				log.Printf("WLAN %q: failed to set: %v", wlan.Name, err)
				continue
			}
			log.Printf("WLAN %q: set enabled=%t", wlan.Name, enable)
		}
	}
}
