package main

import (
	"log"

	"github.com/1Password/connect-sdk-go/connect"
)

type Secrets struct {
	Husqvarna HusqvarnaKeys
	Discord   DiscordKeys
}

func retrieveSecrets() Secrets {
	secrets := Secrets{Husqvarna: HusqvarnaKeys{}, Discord: DiscordKeys{}}
	client, err := connect.NewClientFromEnvironment()
	if err != nil {
		log.Println("Failed to create 1Password client")
		log.Fatal(err)
	}
	item, err := client.GetItem("DISCORD-WEBHOOK-ID", "HomeLab")
	if err != nil {
		log.Fatal(err)
	}
	secrets.Discord.WebhookID = item.GetValue("identifiant")
	item, err = client.GetItem("DISCORD-TOKEN", "HomeLab")
	if err != nil {
		log.Fatal(err)
	}
	secrets.Discord.WebhookToken = item.GetValue("identifiant")
	item, err = client.GetItem("HUSQVARNA-CLIENT-ID", "HomeLab")
	if err != nil {
		log.Fatal(err)
	}
	secrets.Husqvarna.ClientID = item.GetValue("identifiant")
	secrets.Husqvarna.APIKey = secrets.Husqvarna.ClientID
	item, err = client.GetItem("HUSQVARNA-CLIENT-SECRET", "HomeLab")
	if err != nil {
		log.Fatal(err)
	}
	secrets.Husqvarna.ClientSecret = item.GetValue("identifiant")
	log.Println("Successfully retrieved secrets from 1Password")
	return secrets
}
