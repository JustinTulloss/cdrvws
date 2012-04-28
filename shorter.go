package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/simonz05/godis"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

const HELP string = `cdrvws(1)                          CDRV.WS                          cdrvws(1)

NAME
    cdrvws: command line url shortener

SYNOPSIS
    <command> | curl -F 'rvw=<-' http://cdrv.ws

EXAMPLE
    ~$ echo "http://ebushpilot.com/images/polarbear_1.jpg" | curl -F 'rvw=<-' http://cdrv.ws
    http://cdrv.ws/2
    ~$ open http://cdrv.ws/2

AUTHOR
    Justin Tulloss <justin.tulloss at gmail>

SEE ALSO
    http://github.com/JustinTulloss/cdrvws

CREDITS
    Inspired by sprunge: http://github.com/rupa/sprunge`

var redis *godis.Client
var trimmer *regexp.Regexp

func main() {
	connectToRedis()
	http.HandleFunc("/", route)
	trimmer, _ = regexp.Compile("A+=$")
	startServer()
}

func encode(id uint64) string {
	bytes := make([]byte, 8)
	binary.PutUvarint(bytes, id)
	encoded := base64.URLEncoding.EncodeToString([]byte(bytes))
	return trimmer.ReplaceAllLiteralString(encoded, "")
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
		Host: "localhost:6379",
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
	redis = godis.New("tcp:"+parsedurl.Host, 0, password)
}

func startServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting on %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
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
	w.Header().Set("Content-Type", "text/plain")
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
	suffix := ""
	if strings.ContainsRune(shorturl, '/') {
		shorturl, suffix = path.Split(shorturl)
		if shorturl == "" {
			shorturl = suffix
			suffix = ""
		}
	}
	shorturl = strings.Trim(shorturl, "/")
	err, longurl := expand(shorturl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if longurl == "" {
		http.NotFound(w, req)
	} else {
		if !strings.HasSuffix(longurl, "/") {
			longurl += "/"
		}
		http.Redirect(w, req, longurl+suffix, http.StatusMovedPermanently)
	}
}
