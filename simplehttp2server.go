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
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

const (
	PushMarkerHeader = "X-Is-A-Push"
)

var (
	listen   = flag.String("listen", ":5000", "Port to listen on")
	cors     = flag.String("cors", "*", "Set allowed origins")
	firebase = flag.String("firebase", "", "File containing a Firebase static hosting config")
)

func main() {
	flag.Parse()

	server := &http.Server{
		Addr:         *listen,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}
	http2.ConfigureServer(server, &http2.Server{})

	fs := http.FileServer(http.Dir("."))
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", *cors)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTION, HEAD, PATCH, PUT, POST, DELETE")
		log.Printf("Request for %s (Accept-Encoding: %s)", r.URL.Path, r.Header.Get("Accept-Encoding"))

		if *firebase != "" {
			processWithFirebase(w, r, *firebase)
		}
		if r.Header.Get(PushMarkerHeader) == "" {
			pushResources(w, w.Header().Get("Link"))
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

func pushResources(w http.ResponseWriter, linkHeader string) {
	parts := strings.Split(linkHeader, ",")
	pusher, ok := w.(http.Pusher)
	if !ok {
		log.Printf("ResponseWriter is not a Pusher. Not pushing anything")
		return
	}
	for _, part := range parts {
		if !strings.Contains(part, "rel=preload") {
			continue
		}
		resource := extractResourceFromLinkHeader(part)
		pusher.Push(resource, &http.PushOptions{
			Method: "GET",
			Header: http.Header{
				PushMarkerHeader: []string{"true"},
			},
		})
	}
}

var extractionRegexp = regexp.MustCompile("<([^>]+>)")

func extractResourceFromLinkHeader(part string) string {
	return extractionRegexp.FindStringSubmatch(part)[1]
}
