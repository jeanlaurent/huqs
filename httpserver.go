package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
)

func startHttpServer(appSecrets Secrets) {
	r := mux.NewRouter()

	r.HandleFunc("/api/mowers", func(w http.ResponseWriter, r *http.Request) {
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

		json.NewEncoder(w).Encode(mowersData)
	})

	r.HandleFunc("/api/mower/{mowerID}/{action}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mowerID := vars["mowerID"]
		action := vars["action"]

		duration := r.URL.Query().Get("duration")
		if duration == "" {
			duration = "60"
		}

		var payload io.Reader

		switch action {
		case "start":
			payload = strings.NewReader(fmt.Sprintf(`{"data":{"type":"Start","attributes":{"duration":%s}}}`, duration))
			sendDiscordMessage("Je commence à tondre pour "+duration+" minutes", appSecrets.Discord)
		case "park":
			payload = strings.NewReader(fmt.Sprintf(`{"data":{"type":"Park","attributes":{"duration":%s}}}`, duration))
			sendDiscordMessage("Je vais à la station de charge pour "+duration+" minutes", appSecrets.Discord)
		case "pause":
			payload = strings.NewReader(`{"data":{"type":"Pause"}}`)
			sendDiscordMessage("Je fais une pause", appSecrets.Discord)
		case "parkschedule":
			payload = strings.NewReader(`{"data":{"type":"ParkUntilNextSchedule"}}`)
			sendDiscordMessage("Je vais me recharger jusqu'à la prochaine tonte", appSecrets.Discord)
		case "resumeschedule":
			payload = strings.NewReader(`{"data":{"type":"ResumeSchedule"}}`)
			sendDiscordMessage("Je reprends mon planning de tonte", appSecrets.Discord)
		}

		authData := Authenticate(appSecrets.Husqvarna)
		client := &http.Client{}
		req, err := http.NewRequest("POST", "https://api.amc.husqvarna.dev/v1/mowers/"+mowerID+"/actions", payload)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authData.AccessToken))
		req.Header.Add("X-Api-Key", appSecrets.Husqvarna.APIKey)
		req.Header.Add("Authorization-Provider", "husqvarna")
		req.Header.Add("Content-Type", "application/vnd.api+json")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprint(w, string(body))
	})

	r.HandleFunc("/api/mower/{mowerID}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mowerID := vars["mowerID"]

		authData := Authenticate(appSecrets.Husqvarna)
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.amc.husqvarna.dev/v1/mowers/"+mowerID, nil)
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

		fmt.Fprint(w, string(body))
	})

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	staticPath := filepath.Join(basepath, "static")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticPath)))

	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Let's go! on port " + port + " 🚀")
	listenErr := http.ListenAndServe(":"+port, nil)
	if listenErr != nil {
		log.Fatal(listenErr)
	}
}
