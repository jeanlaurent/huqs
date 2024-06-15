package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/alexkappa/mustache"
	"github.com/gorilla/mux"
)

type responseCapture struct {
	http.ResponseWriter
	body bytes.Buffer
}

func (r *responseCapture) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func templaterMiddleWare(next http.Handler, mustache *mustache.Template, pageVariables map[string]string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// exit this function if the content is not html based on the file extension
		if filepath.Ext(request.URL.Path) != ".html" || strings.HasSuffix(request.URL.Path, "/") {
			log.Println(request.URL.Path, "not html")
			next.ServeHTTP(writer, request)
			return
		}
		log.Println(request.URL.Path, "html")
		capture := &responseCapture{ResponseWriter: writer}

		next.ServeHTTP(capture, request)

		err := mustache.ParseBytes(capture.body.Bytes())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		buffer := &bytes.Buffer{}
		err = mustache.Render(buffer, pageVariables)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("before", writer.Header())
		bufferBytes := buffer.Bytes()
		log.Println("Content-Length", len(bufferBytes))
		writer.Header().Set("Content-Length", strconv.Itoa(len(bufferBytes)))
		log.Println(writer.Header())
		_, err = writer.Write(buffer.Bytes())
		if err != nil {
			log.Println(err)
		}

	})
}

func listingMowerHandler(appSecrets Secrets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func mowerActionHandler(appSecrets Secrets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func mowerDetailHandler(appSecrets Secrets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func startHttpServer(appSecrets Secrets) {
	router := mux.NewRouter()

	router.HandleFunc("/api/mowers", listingMowerHandler(appSecrets))
	router.HandleFunc("/api/mower/{mowerID}/{action}", mowerActionHandler(appSecrets))
	router.HandleFunc("/api/mower/{mowerID}", mowerDetailHandler(appSecrets))

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	staticPath := filepath.Join(basepath, "static")

	mustache := mustache.New()
	pageVariables := map[string]string{
		"GOOGLEMAP_API_KEY": appSecrets.GoogleMapApiKey,
		"hello":             "Hello World!",
	}

	router.PathPrefix("/").Handler(templaterMiddleWare(http.FileServer(http.Dir(staticPath)), mustache, pageVariables))

	http.Handle("/", router)

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
