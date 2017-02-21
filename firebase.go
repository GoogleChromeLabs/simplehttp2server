package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type FirebaseManifest struct {
	Public    string `json:"public"`
	Redirects []struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Type        int    `json:"type,omitempty"`
	} `json:"redirects"`
	Rewrites []struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	} `json:"rewrites"`
	Hosting struct {
		Headers []struct {
			Source  string `json:"source"`
			Headers []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"headers"`
	} `json:"hosting"`
}

func (mf FirebaseManifest) processRedirects(w http.ResponseWriter, r *http.Request) (bool, error) {
	for _, redirect := range mf.Redirects {
		pattern, err := CompileExtGlob(redirect.Source)
		if err != nil {
			return false, fmt.Errorf("Invalid redirect extglob %s: %s", redirect.Source, err)
		}
		if pattern.MatchString(r.URL.Path) {
			http.Redirect(w, r, redirect.Destination, redirect.Type)
			return true, nil
		}
	}
	return false, nil
}

func (mf FirebaseManifest) processRewrites(r *http.Request) error {
	for _, rewrite := range mf.Rewrites {
		pattern, err := CompileExtGlob(rewrite.Source)
		if err != nil {
			return fmt.Errorf("Invalid rewrite extglob %s: %s", rewrite.Source, err)
		}
		if pattern.MatchString(r.URL.Path) {
			r.URL.Path = rewrite.Destination
			return nil
		}
	}

	return nil
}

func (mf FirebaseManifest) processHosting(w http.ResponseWriter, r *http.Request) error {
	for _, headerSet := range mf.Hosting.Headers {
		pattern, err := CompileExtGlob(headerSet.Source)
		if err != nil {
			return fmt.Errorf("Invalid hosting.header extglob %s: %s", headerSet.Source, err)
		}
		if pattern.MatchString(r.URL.Path) {
			for _, header := range headerSet.Headers {
				w.Header().Add(header.Key, header.Value)
			}
			return nil
		}
	}
	return nil
}

func processWithFirebase(w http.ResponseWriter, r *http.Request, firebaseFile string) string {
	dir := "."
	mf, err := readManifest(firebaseFile)
	if err != nil {
		log.Printf("Could read Firebase file %s: %s", firebaseFile, err)
		return dir
	}
	if mf.Public != "" {
		dir = mf.Public
	}

	done, err := mf.processRedirects(w, r)
	if err != nil {
		log.Printf("Processing redirects failed: %s", err)
		return dir
	}
	if done {
		return dir
	}

	err = mf.processRewrites(r)
	if err != nil {
		log.Printf("Processing rewrites failed: %s", err)
		return dir
	}

	err = mf.processHosting(w, r)
	if err != nil {
		log.Printf("Processing rewrites failed: %s", err)
		return dir
	}
	return dir
}

func readManifest(path string) (FirebaseManifest, error) {
	fmf := FirebaseManifest{}
	f, err := os.Open(path)
	if err != nil {
		return fmf, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = dec.Decode(&fmf)
	return fmf, err
}
