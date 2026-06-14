package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alexkappa/mustache"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func listingMowerHandler(appSecrets Secrets) echo.HandlerFunc {
	return func(c echo.Context) error {
		mowersData, err := getMowerStatus(appSecrets.Husqvarna)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, mowersData)
	}
}

func mowerActionHandler(appSecrets Secrets) echo.HandlerFunc {
	return func(c echo.Context) error {
		mowerID := c.Param("mowerID")
		action := c.Param("action")

		duration := c.QueryParam("duration")
		if duration == "" {
			duration = "60"
		}

		var payload io.Reader
		var err error
		switch action {
		case "start":
			payload = strings.NewReader(fmt.Sprintf(`{"data":{"type":"Start","attributes":{"duration":%s}}}`, duration))
			err = sendDiscordMessage("Je commence à tondre pour "+duration+" minutes", appSecrets.Discord)
		case "park":
			payload = strings.NewReader(fmt.Sprintf(`{"data":{"type":"Park","attributes":{"duration":%s}}}`, duration))
			err = sendDiscordMessage("Je vais à la station de charge pour "+duration+" minutes", appSecrets.Discord)
		case "pause":
			payload = strings.NewReader(`{"data":{"type":"Pause"}}`)
			err = sendDiscordMessage("Je fais une pause", appSecrets.Discord)
		case "parkschedule":
			payload = strings.NewReader(`{"data":{"type":"ParkUntilNextSchedule"}}`)
			err = sendDiscordMessage("Je vais me recharger jusqu'à la prochaine tonte", appSecrets.Discord)
		case "resumeschedule":
			payload = strings.NewReader(`{"data":{"type":"ResumeSchedule"}}`)
			err = sendDiscordMessage("Je reprends mon planning de tonte", appSecrets.Discord)
		}
		if err != nil {
			return err
		}

		authData := Authenticate(appSecrets.Husqvarna)
		client := &http.Client{}
		req, err := http.NewRequest("POST", "https://api.amc.husqvarna.dev/v1/mowers/"+mowerID+"/actions", payload)
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authData.AccessToken))
		req.Header.Add("X-Api-Key", appSecrets.Husqvarna.APIKey)
		req.Header.Add("Authorization-Provider", "husqvarna")
		req.Header.Add("Content-Type", "application/vnd.api+json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return c.HTML(http.StatusOK, string(body))
	}
}

func mowerDetailHandler(appSecrets Secrets) echo.HandlerFunc {
	return func(c echo.Context) error {
		mowerID := c.Param("mowerID")

		authData := Authenticate(appSecrets.Husqvarna)
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.amc.husqvarna.dev/v1/mowers/"+mowerID, nil)
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authData.AccessToken))
		req.Header.Add("X-Api-Key", appSecrets.Husqvarna.APIKey)
		req.Header.Add("Authorization-Provider", "husqvarna")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return c.HTML(http.StatusOK, string(body))
	}
}

func findStaticPath() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "static")
}

func findFileToServe(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		file = filepath.Join(file, "index.html")
	}
	return file, nil
}

func mustacheMe(i *echo.Echo, prefix, root string, mustache *mustache.Template, pageVariables map[string]string) *echo.Route {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}
	h := func(c echo.Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}
		name := filepath.Join(root, path.Clean("/"+p)) // "/"+ for security
		fileToServe, err := findFileToServe(name)
		if err != nil {
			return err
		}
		if filepath.Ext(fileToServe) == ".html" {
			file, err := os.Open(fileToServe)
			if err != nil {
				return err
			}
			defer file.Close()
			err = mustache.Parse(file)
			if err != nil {
				return err
			}
			content, err := mustache.RenderString(pageVariables)
			if err != nil {
				return err
			}
			return c.HTML(http.StatusOK, content)
		}
		return c.File(name)
	}
	i.GET(prefix, h)
	if prefix == "/" {
		return i.GET(prefix+"*", h)
	}
	return i.GET(prefix+"/*", h)
}

func listLastMessages(c echo.Context) error {
	return c.JSON(http.StatusOK, queue.GetLast100Messages())
}

func startHttpServer(appSecrets Secrets) {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/api/mowers", listingMowerHandler(appSecrets))
	e.GET("/api/mower/:mowerID/:action", mowerActionHandler(appSecrets))
	e.GET("/api/mower/:mowerID", mowerDetailHandler(appSecrets))
	e.GET("/api/messages", listLastMessages)

	// I used to be serving file with
	// e.Static("/", findStaticPath())

	// This is a pretty dirty but working implementation
	// Of serving html files with mustache templating
	pageVariables := map[string]string{
		"GOOGLEMAPAPIKEY": appSecrets.GoogleMapApiKey,
		"hello":           "Hello World!",
	}
	mustacheMe(e, "/", findStaticPath(), mustache.New(), pageVariables)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Let's go! on port " + port + " 🚀")
	listenErr := e.Start(":" + port)
	if listenErr != nil {
		log.Fatal(listenErr) // we purposely crash the app here
	}

}
