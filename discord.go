package main

import (
	"github.com/DisgoOrg/disgohook"
)

type DiscordKeys struct {
	WebhookID    string
	WebhookToken string
}

func sendDiscordMessage(content string, keys DiscordKeys) error {
	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, keys.WebhookID+"/"+keys.WebhookToken)
	if err != nil {
		return err
	}
	_, err = webhook.SendContent(content)
	if err != nil {
		return err
	}
	return nil
}
