// Webik - Micro web server to run Frontend applications based on Angular, Vue, React, etc.

// - Handling HTTP GET "/" "" for index.html
// - Processing GET for files with extensions: css, html, js
//- Create reverse proxy for the backend
// Written by Andrey Simanov 2021

package webik

import (
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	Port           string
	SourceApiRoute string
	TargetApiRoute string
	Path           string
}

var opts options


func ListenAndServe(Port string, SitePath string, TargetApiRoute string, SourceApiRoute string) {

	err := mime.AddExtensionType(".css", "text/css")
	if err == nil {
		err = mime.AddExtensionType(".js", "text/javascript")
	}
	if err == nil {
		err = mime.AddExtensionType(".html", "text/html; charset=utf-8")
	}
	if err != nil {
		log.Printf("Warning: unable to register extension types %v", err)
	}

	opts = options{
		Port:           Port,
		Path:           SitePath,
		SourceApiRoute: SourceApiRoute,
		TargetApiRoute: TargetApiRoute,
	}

	http.HandleFunc("/", requestHandler)
	log.Println("Webik Listen on port", opts.Port+"...")
	err = http.ListenAndServe(opts.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	mainPath, _ := os.Getwd()
	path := r.URL.Path
	if opts.SourceApiRoute != "" && strings.Index(path, opts.SourceApiRoute) == 0 {
		proxyHandler(w, r, opts.TargetApiRoute, opts.SourceApiRoute)
		return
	}
	switch r.Method {
	case "GET":
		fileExt := filepath.Ext(path)
		if path == "" || path == "/" || fileExt == "" {
			path = "/index.html"
		}
		filePath := strings.Replace(path, "/", string(os.PathSeparator), -1)
		content, err := ioutil.ReadFile(mainPath + string(os.PathSeparator) + opts.Path + string(os.PathSeparator) + filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		contentType := mime.TypeByExtension(fileExt)
		w.Header().Set("Content-Type", contentType)
		_, err = w.Write(content)
		if err != nil {
			log.Printf("Warning: unable to write response: %v", err)
		}
	}
}


func proxyHandler(res http.ResponseWriter, req *http.Request, TargetUrl string, SrcUrl string) {
	targetUrl, _ := url.Parse(TargetUrl)
	log.Printf("Target URL: %s", targetUrl)
	log.Printf("Source Request: %v", req)
	req.URL.Host = targetUrl.Host
	req.URL.Scheme = targetUrl.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.URL, _ = url.Parse(strings.Replace(req.URL.String(), SrcUrl, "", 1))
	req.RequestURI = strings.Replace(req.RequestURI, SrcUrl, "", 1)
	req.Host = targetUrl.Host
	log.Printf("Request: %v", req)
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.ServeHTTP(res, req)
}
