// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"mime"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
)

const (
	PushMarkerHeader = "X-Is-A-Push"
)

var (
	listen = flag.String("listen", ":5000", "Port to listen on")
	cors   = flag.String("cors", "*", "Set allowed origins")
	config = flag.String("config", "", "Config file")
)

func main() {
	flag.Parse()

	server := &http.Server{
		Addr:         *listen,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		TLSConfig: &tls.Config{
			NextProtos:               []string{"h2", "h2-14"},
			PreferServerCipherSuites: true,
		},
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", *cors)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTION, HEAD, PATCH, PUT, POST, DELETE")
		log.Printf("Request for %s (Accept-Encoding: %s)", r.URL.Path, r.Header.Get("Accept-Encoding"))

		dir := "."
		redirected := false
		if *config != "" {
			dir, redirected = processWithConfig(w, r, *config)
		}
		if redirected {
			return
		}
		if r.Header.Get(PushMarkerHeader) == "" {
			pushResources(w)
		}

		// Add GZIP compression if it is a text-based format
		fs := http.FileServer(http.Dir(dir))
		typ := mime.TypeByExtension(r.URL.Path)
		switch {
		case strings.HasPrefix(typ, "text/"):
		case typ == "application/xml":
		case typ == "":
			fs = gziphandler.GzipHandler(fs)
		}

		fs.ServeHTTP(w, r)
	})

	if err := configureTLS(server); err != nil {
		log.Fatalf("Error configuring TLS: %s", err)
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatalf("Error opening socket: %s", err)
	}
	ln = &HijackHTTPListener{ln}

	tlsListener := tls.NewListener(ln, server.TLSConfig)
	tcl := tlsListener
	if strings.HasPrefix(*listen, ":") {
		*listen = "localhost" + *listen
	}
	log.Printf("Listening on https://%s...", *listen)
	if err := server.Serve(tcl); err != nil {
		log.Fatalf("Error starting webserver: %s", err)
	}
}

func pushResources(w http.ResponseWriter) {
	linkHeader := w.Header().Get("Link")
	parts := strings.Split(linkHeader, ",")
	pusher, ok := w.(http.Pusher)
	if !ok {
		log.Printf("ResponseWriter is not a Pusher. Not pushing anything")
		return
	}
	newParts := []string{}
	for _, part := range parts {
		if !strings.Contains(part, "rel=preload") {
			newParts = append(newParts, part)
			continue
		}
		resource := extractResourceFromLinkHeader(part)
		if !strings.HasPrefix(resource, "/") {
			log.Printf("--> Push attempt: Resource path needs to start with /")
			continue
		}
		log.Printf("--> Push: %s", resource)
		pusher.Push(resource, &http.PushOptions{
			Method: "GET",
			Header: http.Header{
				PushMarkerHeader: []string{"true"},
			},
		})
	}
	w.Header().Set("Link", strings.Join(newParts, ","))
}

var extractionRegexp = regexp.MustCompile("<([^>]+)>")

func extractResourceFromLinkHeader(part string) string {
	return extractionRegexp.FindStringSubmatch(part)[1]
}
