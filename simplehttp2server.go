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
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/efarrer/iothrottler"

	// This is a temporary import!
	// It is the same as golang.org/x/net/http2 with a patch by brk0v
	// to expose the push functionality.
	"github.com/GoogleChrome/simplehttp2server/http2"
)

type PushResponseWriter struct {
	http.ResponseWriter
	http2.Pusher
}

var (
	listen       = flag.String("listen", ":5000", "Port to listen on")
	http1        = flag.Bool("http1", false, "Serve via HTTP/1.1")
	disableGzip  = flag.Bool("nogzip", false, "Disable GZIP content compression")
	cors         = flag.String("cors", "*", "Set allowed origins")
	pushManifest = flag.String("pushmanifest", "push.json", "File containing the push manifest")
	minDelay     = flag.Int("mindelay", 0, "Minimum delay before a request in answered in milliseconds (ignored without -maxdelay)")
	maxDelay     = flag.Int("maxdelay", 0, "Maximum delay before a request in answered in milliseconds")
	spa          = flag.String("spa", "", "Page to serve instead of 404")
	throttle     = flag.Int("throttle", 0, "Maximum throughput of network in kbps (0 = no limit)")
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

	if !*http1 {
		http2.ConfigureServer(server, &http2.Server{})
	}

	fs := http.FileServer(http.Dir("."))
	if !*disableGzip {
		oldfs := fs
		fs = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			grw := GzipResponseWriter{gzip.NewWriter(w), w}
			oldfs.ServeHTTP(grw, r)
			grw.WriteCloser.Close()
		})
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", *cors)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTION, HEAD, PATCH, PUT, POST, DELETE")
		log.Printf("Request for %s (Accept-Encoding: %s)", r.URL.Path, r.Header.Get("Accept-Encoding"))

		if *throttle != 0 {
			trw := NewThrottledResponseWriter(w, iothrottler.Bandwidth(*throttle)*iothrottler.Kbps)
			defer trw.Free()
			w = trw
		}

		delay := 0
		if *maxDelay > *minDelay {
			delay = mrand.Intn(*maxDelay-*minDelay) + *minDelay
		}
		if queryDelay := r.FormValue("delay"); queryDelay != "" {
			delay, _ = strconv.Atoi(queryDelay)
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)

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

		if !*http1 && r.Header.Get("X-Is-Push") != "true" {
			pushResources(w, r)
		}
	})

	if err := configureTLS(server); err != nil {
		log.Fatalf("Error configuring TLS: %s", err)
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatalf("Error opening socket: %s", err)
	}

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
	pusher, ok := w.(http2.Pusher)
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
		// Add a X-Header to the pushed request so we don’t trigger pushes for pushes
		pusher.Push("GET", key, http.Header{
			"X-Is-Push": []string{"true"},
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

type GzipResponseWriter struct {
	io.WriteCloser
	http.ResponseWriter
}

func (gzw GzipResponseWriter) Write(b []byte) (int, error) {
	return gzw.WriteCloser.Write(b)
}

type ThrottledResponseWriter struct {
	http2.Pusher
	io.Writer
	http.ResponseWriter
	p *iothrottler.IOThrottlerPool
}

func (trw ThrottledResponseWriter) Write(b []byte) (int, error) {
	return trw.Writer.Write(b)
}

func (trw ThrottledResponseWriter) Free() {
	trw.p.ReleasePool()
}

func NewThrottledResponseWriter(w http.ResponseWriter, bw iothrottler.Bandwidth) ThrottledResponseWriter {
	p := iothrottler.NewIOThrottlerPool(bw)
	tw, _ := p.AddWriter(NopWriteCloser{w})
	pusher, _ := w.(http2.Pusher)
	return ThrottledResponseWriter{pusher, tw, w, p}
}

type NopWriteCloser struct {
	io.Writer
}

func (nwc NopWriteCloser) Close() error {
	return nil
}

var (
	validFrom  = time.Now()
	validFor   = 365 * 24 * time.Hour
	isCA       = true
	rsaBits    = 2048
	ecdsaCurve = ""
)

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func generateCertificates(host string) {
	var priv interface{}
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	var notBefore = validFrom
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Print("written cert.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return
	}
	pem.Encode(keyOut, pemBlockForKey(priv))
	keyOut.Close()
	log.Print("written key.pem\n")
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

func configureTLS(server *http.Server) error {
	if _, err := os.Stat("cert.pem"); err != nil {
		log.Printf("Generating certificate...")
		generateCertificates("localhost")
	}

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		return err
	}

	if server.TLSConfig == nil {
		server.TLSConfig = &tls.Config{}
	}
	server.TLSConfig.PreferServerCipherSuites = true
	server.TLSConfig.NextProtos = append(server.TLSConfig.NextProtos, "http/1.1")
	server.TLSConfig.Certificates = []tls.Certificate{cert}
	return nil
}
