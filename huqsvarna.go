package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var authData AuthResponse
var once sync.Once

type HusqvarnaKeys struct {
	ClientID     string
	APIKey       string
	ClientSecret string
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	Provider    string `json:"provider"`
	UserID      string `json:"user_id"`
	TokenType   string `json:"token_type"`
}

type MowerAction struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Duration int `json:"duration"`
		} `json:"attributes"`
	} `json:"data"`
}

type MowersResponse struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			System struct {
				Name         string `json:"name"`
				Model        string `json:"model"`
				SerialNumber int    `json:"serialNumber"`
			} `json:"system"`
			Battery struct {
				BatteryPercent int `json:"batteryPercent"`
			} `json:"battery"`
			Capabilities struct {
				Headlights   bool `json:"headlights"`
				WorkAreas    bool `json:"workAreas"`
				Position     bool `json:"position"`
				StayOutZones bool `json:"stayOutZones"`
			} `json:"capabilities"`
			Mower struct {
				Mode               string `json:"mode"`
				Activity           string `json:"activity"`
				InactiveReason     string `json:"inactiveReason"`
				State              string `json:"state"`
				ErrorCode          int    `json:"errorCode"`
				ErrorCodeTimestamp int    `json:"errorCodeTimestamp"`
			} `json:"mower"`
			Calendar struct {
				Tasks []struct {
					Start      int  `json:"start"`
					Duration   int  `json:"duration"`
					Monday     bool `json:"monday"`
					Tuesday    bool `json:"tuesday"`
					Wednesday  bool `json:"wednesday"`
					Thursday   bool `json:"thursday"`
					Friday     bool `json:"friday"`
					Saturday   bool `json:"saturday"`
					Sunday     bool `json:"sunday"`
					WorkAreaID int  `json:"workAreaId"`
				} `json:"tasks"`
			} `json:"calendar"`
			Planner struct {
				NextStartTimestamp int64 `json:"nextStartTimestamp"`
				Override           struct {
					Action string `json:"action"`
				} `json:"override"`
				RestrictedReason string `json:"restrictedReason"`
			} `json:"planner"`
			Metadata struct {
				Connected       bool  `json:"connected"`
				StatusTimestamp int64 `json:"statusTimestamp"`
			} `json:"metadata"`
			WorkAreas []struct {
				WorkAreaID    int    `json:"workAreaId"`
				Name          string `json:"name"`
				CuttingHeight int    `json:"cuttingHeight"`
			} `json:"workAreas"`
			Positions []struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"positions"`
			Settings struct {
				CuttingHeight int `json:"cuttingHeight"`
				Headlight     struct {
					Mode string `json:"mode"`
				} `json:"headlight"`
			} `json:"settings"`
			Statistics struct {
				CuttingBladeUsageTime  int `json:"cuttingBladeUsageTime"`
				NumberOfChargingCycles int `json:"numberOfChargingCycles"`
				NumberOfCollisions     int `json:"numberOfCollisions"`
				TotalChargingTime      int `json:"totalChargingTime"`
				TotalCuttingTime       int `json:"totalCuttingTime"`
				TotalDriveDistance     int `json:"totalDriveDistance"`
				TotalRunningTime       int `json:"totalRunningTime"`
				TotalSearchingTime     int `json:"totalSearchingTime"`
			} `json:"statistics"`
			StayOutZones struct {
				Zones []interface{} `json:"zones"`
				Dirty bool          `json:"dirty"`
			} `json:"stayOutZones"`
		} `json:"attributes"`
	} `json:"data"`
}

func activityMessage(activity string) string {
	activityDescriptions := map[string]string{
		"UNKNOWN":           "Inconnu.",
		"NOT_APPLICABLE":    "Non applicable.",
		"MOWING":            "En train de tondre.",
		"GOING_HOME":        "Rentre à la station de charge.",
		"CHARGING":          "En train de charger.",
		"LEAVING":           "Quitte actuellement la station de charge et se dirige vers un point de départ.",
		"PARKED_IN_CS":      "Garée dans la station de charge.",
		"STOPPED_IN_GARDEN": "Est arrêtée dans le jardin.",
	}

	return activityDescriptions[activity]
}

func huqsvarnaAuthenticate(keys HusqvarnaKeys) AuthResponse {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", keys.ClientID)
	data.Set("client_secret", keys.ClientSecret)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.authentication.husqvarnagroup.dev/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var authData AuthResponse
	err = json.Unmarshal(body, &authData)
	if err != nil {
		log.Fatal(err)
	}

	return authData
}

func Authenticate(keys HusqvarnaKeys) AuthResponse {
	once.Do(func() {
		authData = huqsvarnaAuthenticate(keys)
	})
	return authData
}

func checkMowerStatus(appSecrets Secrets) {
	authData := Authenticate(appSecrets.Husqvarna)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.amc.husqvarna.dev/v1/mowers", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authData.AccessToken))
	req.Header.Add("X-Api-Key", appSecrets.Husqvarna.APIKey)
	req.Header.Add("Authorization-Provider", "husqvarna")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var mowersData MowersResponse
	err = json.Unmarshal(body, &mowersData)
	if err != nil {
		log.Fatal(err)
	}
	newActivity := mowersData.Data[0].Attributes.Mower.Activity
	fmt.Println(time.Now().Format("15:04:05"), "Comparing activity: ", mowerActivity, " vs ", newActivity)
	if mowerActivity != newActivity {
		sendDiscordMessage(activityMessage(newActivity), appSecrets.Discord)
		mowerActivity = newActivity
	}

}
