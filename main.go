package main

import (
	"log"

	"github.com/go-co-op/gocron/v2"
)

var mowerActivity string

func startCron(appSecrets Secrets) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}
	_, _ = scheduler.NewJob(
		gocron.CronJob("*/1 * * * *", false),
		gocron.NewTask(checkMowerStatus, appSecrets),
	)
	if err != nil {
		panic(err) // really? panic?
	}

	scheduler.Start()
}

func main() {
	appSecrets, err := retrieveSecrets()
	if err != nil {
		log.Fatal(err) // we crash here if we can't retrieve secrets
	}

	startCron(appSecrets)

	startHttpServer(appSecrets)
}
