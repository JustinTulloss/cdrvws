package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "net/http"
    "net/url"
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

var shortid uint64 = 1 // We don't allow an id at 0
var shorturls map[string] string

func main() {
    shorturls = make(map[string] string)
    http.HandleFunc("/", route)
    startServer()
}

func encode(id uint64) string {
    encoded := ""
    for id > 0 {
        encoded += string(CHARS[id % BASE])
        id = id / BASE
    }
    return encoded
}

func createShortUrl(longurl string) (error, string) {
    shorturl := encode(shortid)
    shortid++
    shorturls[shorturl] = longurl
    return nil, shorturl
}

func expand(shorturl string) (error, string) {
    longurl, _ := shorturls[shorturl]
    return nil, longurl
}

func startServer() {
    err := http.ListenAndServe(":" + os.Getenv("PORT"), nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}

func route(w http.ResponseWriter, req *http.Request) {
    if (req.Method == "GET") {
        if (req.URL.String() == "/") {
            handleHome(w, req)
        } else {
            handleExpand(w, req)
        }
    } else if (req.Method == "POST") {
        handleShorten(w, req)
    }
}

func handleHome(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintln(w, HELP)
}

func handleShorten(w http.ResponseWriter, req *http.Request) {
    err, shorturl := createShortUrl(req.FormValue("rvw"))
    if (err != nil) {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fullurl := url.URL{
        Scheme: "http",
        Host: req.Host,
        Path: "/" + shorturl,
    }
    fmt.Fprintln(w, fullurl.String())
}

func handleExpand(w http.ResponseWriter, req *http.Request) {
    shorturl := strings.Trim(req.URL.String(), "/")
    err, longurl := expand(shorturl)
    if (err != nil) {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    } else if (longurl == "") {
        http.NotFound(w, req)
    } else {
        http.Redirect(w, req, longurl, http.StatusMovedPermanently)
    }
}

