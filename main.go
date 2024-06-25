package main

import (
	"log"
)

func main() {
	log.Println("Huqs starting...")

	appSecrets, err := retrieveSecrets()
	if err != nil {
		log.Fatal(err) // we crash here if we can't retrieve secrets
	}

	startCron(appSecrets)

	startHttpServer(appSecrets)
}
