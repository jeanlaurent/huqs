package main

import (
	"log"

	"github.com/go-co-op/gocron/v2"
)

var mowerActivity string
var queue *MessageQueue

func startCron(appSecrets Secrets) error {
	queue = NewMessageQueue(100)

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	_, err = scheduler.NewJob(
		gocron.CronJob("*/1 * * * *", false),
		gocron.NewTask(checkMowerStatus, appSecrets, queue),
	)
	if err != nil {
		return err
	}
	log.Println("Cron starting... ")

	scheduler.Start()

	return nil
}
