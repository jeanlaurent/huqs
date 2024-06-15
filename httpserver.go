package main

import (
	"encoding/json"
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
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// type responseCapture struct {
// 	http.ResponseWriter
// 	body bytes.Buffer
// }

// func (r *responseCapture) Write(b []byte) (int, error) {
// 	return r.body.Write(b)
// }

// func templaterMiddleWare(next http.Handler, mustache *mustache.Template, pageVariables map[string]string) http.Handler {
// 	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

// 		if filepath.Ext(request.URL.Path) != ".html" || strings.HasSuffix(request.URL.Path, "/") {
// 			log.Println(request.URL.Path, "not html")
// 			next.ServeHTTP(writer, request)
// 			return
// 		}
// 		log.Println(request.URL.Path, "html")
// 		capture := &responseCapture{ResponseWriter: writer}

// 		next.ServeHTTP(capture, request)

// 		err := mustache.ParseBytes(capture.body.Bytes())
// 		if err != nil {
// 			http.Error(writer, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		buffer := &bytes.Buffer{}
// 		err = mustache.Render(buffer, pageVariables)
// 		if err != nil {
// 			http.Error(writer, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 		log.Println("before", writer.Header())
// 		bufferBytes := buffer.Bytes()
// 		log.Println("Content-Length", len(bufferBytes))
// 		writer.Header().Set("Content-Length", strconv.Itoa(len(bufferBytes)))
// 		log.Println(writer.Header())
// 		_, err = writer.Write(buffer.Bytes())
// 		if err != nil {
// 			log.Println(err)
// 		}

// 	})
// }

func listingMowerHandler(appSecrets Secrets) echo.HandlerFunc {
	return func(c echo.Context) error {
		authData := Authenticate(appSecrets.Husqvarna)

		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://api.amc.husqvarna.dev/v1/mowers", nil)
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

		var mowersData MowersResponse
		err = json.Unmarshal(body, &mowersData)
		if err != nil {
			return err
		}

		c.JSON(http.StatusOK, mowersData)
		return nil
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
		c.HTML(http.StatusOK, string(body))
		return nil
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

		c.HTML(http.StatusOK, string(body))
		return nil
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

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, "index.html")
		return file, nil
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
		} else {
			return c.File(name)
		}
	}
	i.GET(prefix, h)
	if prefix == "/" {
		return i.GET(prefix+"*", h)
	}
	return i.GET(prefix+"/*", h)
}

func startHttpServer(appSecrets Secrets) {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/api/mowers", listingMowerHandler(appSecrets))
	e.GET("/api/mower/:mowerID/:action", mowerActionHandler(appSecrets))
	e.GET("/api/mower/:mowerID", mowerDetailHandler(appSecrets))

	// I used to be serving file with
	// e.Static("/", findStaticPath())
	// This is a pretty dirty but working implementation
	// Of serving html files with mustache templating
	mustache := mustache.New()
	pageVariables := map[string]string{
		"GOOGLEMAPAPIKEY": appSecrets.GoogleMapApiKey,
		"hello":           "Hello World!",
	}
	mustacheMe(e, "/", findStaticPath(), mustache, pageVariables)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Let's go! on port " + port + " 🚀")
	listenErr := e.Start(":" + port)
	if listenErr != nil {
		log.Fatal(listenErr)
	}

	// router.PathPrefix("/").Handler(templaterMiddleWare(http.FileServer(http.Dir(staticPath)), mustache, pageVariables))

	// http.Handle("/", router)
}
