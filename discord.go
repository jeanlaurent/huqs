package main

import (
	"log"

	"github.com/DisgoOrg/disgohook"
)

type DiscordKeys struct {
	WebhookID    string
	WebhookToken string
}

func sendDiscordMessage(content string, keys DiscordKeys) {
	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, keys.WebhookID+"/"+keys.WebhookToken)
	if err != nil {
		log.Fatal(err)
	}
	_, err = webhook.SendContent(content)
	if err != nil {
		log.Fatal(err)
	}
}
