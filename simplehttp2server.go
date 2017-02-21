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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	listen       = flag.String("listen", ":5000", "Port to listen on")
	cors         = flag.String("cors", "*", "Set allowed origins")
	pushManifest = flag.String("pushmanifest", "push.json", "File containing the push manifest")
	spa          = flag.String("spa", "", "Page to serve instead of 404")
)

func init() {
	mrand.Seed(time.Now().Unix())
}

func main() {
	flag.Parse()

	server := &http.Server{
		Addr:         *listen,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}

	fs := http.FileServer(http.Dir("."))
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect and avoid recursive pushes
		if r.Header.Get("X-Is-Push") != "true" {
			pushResources(w, r)
		}

		w.Header().Set("Access-Control-Allow-Origin", *cors)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTION, HEAD, PATCH, PUT, POST, DELETE")
		log.Printf("Request for %s (Accept-Encoding: %s)", r.URL.Path, r.Header.Get("Accept-Encoding"))

		if *spa != "" {
			path := r.URL.Path
			if _, err := os.Stat("." + path); err == nil {
				fs.ServeHTTP(w, r)
			} else {
				spaContents, err := readSPAFile(*spa)
				if err != nil {
					http.Error(w, fmt.Sprintf("Could not read SPA file: %s", err), http.StatusInternalServerError)
					return
				}
				w.Write(spaContents)
			}
		} else {
			fs.ServeHTTP(w, r)
		}

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

func pushResources(w http.ResponseWriter, r *http.Request) {
	pushMap, err := readPushMap(*pushManifest)
	if err != nil {
		log.Printf("Could not load push manifest \"%s\": %s", *pushManifest, err)
		return
	}
	pushes, ok := pushMap[r.URL.Path]
	if !ok {
		log.Printf("No pushes defined for %s", r.URL.Path)
		return
	}
	pusher, ok := w.(http.Pusher)
	if !ok {
		log.Printf("Connection is not a pusher")
		return
	}
	for key, pushInstruction := range pushes {
		_ = pushInstruction // No use just yet
		if key[0] != '/' {
			log.Printf("Keys in the push manifest must start with '/'")
			continue
		}
		log.Printf("Pushing %s", key)
		pusher.Push(key, &http.PushOptions{
			Method: "GET",
			Header: http.Header{
				// Add a X-Header to the pushed request so we can avoid recursive pushing
				"X-Is-Push": []string{"true"},
			},
		})
	}
}

func readSPAFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

type PushManifest map[string]map[string]PushInstruction
type PushInstruction struct {
	Type   string `json:"style"`
	Weight int    `json:"weight"`
}

func readPushMap(filename string) (pm PushManifest, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = dec.Decode(&pm)
	return
}
