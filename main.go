package main

import (
	"log"
	"sync"

	"github.com/go-co-op/gocron/v2"
)

type Secrets struct {
	Husqvarna HusqvarnaKeys
	Discord   DiscordKeys
}

var authData AuthResponse
var once sync.Once

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
		panic(err)
	}

	scheduler.Start()
}

func main() {
	appSecrets := retrieveSecrets()

	startCron(appSecrets)

	startHttpServer(appSecrets)
}
