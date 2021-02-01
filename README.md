# Webik

Webik - Micro web server for running Frontend application locally based on Angular, Vue, React, etc.

Handling HTTP GET "/" "" for index.html
Processing GET for files with extensions: css, html, js, etc
Create reverse proxy for the backend

##License: 
Mit license
 


##Example

``` go
package main

import (
	"github/aas-spec/webik"
	"flag"
	"log"
)

var serverPort = flag.String("serverPort", ":5050", "Server Port")
var sitePath = flag.String("sitePath", "./site", "Site Path")
var sourceRoute = flag.String("sourceRoute", "", "Source Api Route")
var targetRoute = flag.String("targetRoute", "", "Target Api Route")


func main() {
	flag.Parse()
	log.Print("Mini web server, Webik 2021")
	log.Printf("Site Path: %s", *sitePath)
	if *sourceRoute != "" {
		log.Printf("Source Route: %s, Target Route: %s", *sourceRoute, *targetRoute)
	}
	webik.ListenAndServe(*serverPort, *sitePath, *targetRoute, *sourceRoute)
}
```

##Command Line Arguments

Usage: weserver --serverPort=:5050 --sitePath=./mysappite --sourceRoute=/api" --targetRoute=http:/127.0.0.1:5555/api

serverPort - WebServer Port 
sitePath - The path to a web application on your hard drive 
sourceRoute - The part of API url as Frontend knows it 
targetRoute - The initial part of Api url