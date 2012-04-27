package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"github.com/simonz05/godis"
)

const HELP string = `
<!DOCTYPE html>
<html>
    <head>
        <title>cdrv.ws</title>
    </head>
    <body>
        <pre>
cdrvws(1)                          CDRV.WS                          cdrvws(1)

NAME
    cdrvws: command line url shortener:

SYNOPSIS
    &lt;command&gt; | curl -F 'rvw=<-' http://cdrv.ws
        </pre>
    </body>
</html>`

const CHARS string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const BASE uint64 = uint64(len(CHARS))

var redis *godis.Client

func main() {
	connectToRedis()
	http.HandleFunc("/", route)
	startServer()
}

func encode(id uint64) string {
	encoded := ""
	for id > 0 {
		encoded += string(CHARS[id%BASE])
		id = id / BASE
	}
	return encoded
}

func createShortUrl(longurl string) (error, string) {
	shortid, err := redis.Incr("urlId")
	if err != nil {
		return err, ""
	}
	shorturl := encode(uint64(shortid))
	if err := redis.Set(shorturl, longurl); err != nil {
		return err, ""
	}
	return nil, shorturl
}

func expand(shorturl string) (error, string) {
	longurl, err := redis.Get(shorturl)
	if err != nil && err.Error() == "Nonexisting key" {
		return nil, ""
	} else if err != nil {
		return err, ""
	}
	return nil, longurl.String()
}

func connectToRedis() {
	rawurl := os.Getenv("REDISTOGO_URL")
	log.Printf("Redis to go url: %s\n", rawurl)
	redisurl := url.URL{
		User: url.UserPassword("", ""),
	}
	parsedurl := &redisurl
	if rawurl != "" {
		var err error
		parsedurl, err = parsedurl.Parse(rawurl)
		if err != nil {
			log.Fatal("Could not parse redis url", err)
		}
	}
	password, _ := parsedurl.User.Password()
	log.Printf("Connecting to redis: '%s' with password '%s'\n", parsedurl.Host, password)
	redis = godis.New(parsedurl.Host, 0, password)
}

func startServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting on %s\n", port)
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func route(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		if req.URL.String() == "/" {
			handleHome(w, req)
		} else {
			handleExpand(w, req)
		}
	} else if req.Method == "POST" {
		handleShorten(w, req)
	}
}

func handleHome(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, HELP)
}

func handleShorten(w http.ResponseWriter, req *http.Request) {
	err, shorturl := createShortUrl(req.FormValue("rvw"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fullurl := url.URL{
		Scheme: "http",
		Host:   req.Host,
		Path:   "/" + shorturl,
	}
	fmt.Fprintln(w, fullurl.String())
}

func handleExpand(w http.ResponseWriter, req *http.Request) {
	shorturl := strings.Trim(req.URL.String(), "/")
	err, longurl := expand(shorturl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if longurl == "" {
		http.NotFound(w, req)
	} else {
		http.Redirect(w, req, longurl, http.StatusMovedPermanently)
	}
}
