package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Route struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
	Active bool
}

type Config struct {
	File   string `json:"-"`
	Port   int
	Routes []Route
	mu     sync.RWMutex
}

func (c *Config) Serve(cert, key string) error {
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		source := strings.ToLower(r.Host)

		c.mu.RLock()
		dest, err := findDestination(c.Routes, source)
		c.mu.RUnlock()
		if err != nil {
			http.Error(w, "Host not found", http.StatusNotFound)
			return
		}

		target, err := url.Parse(dest)
		if err != nil {
			http.Error(w, "Invalid URL format", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ServeHTTP(w, r)
	}

	if cert != "" && key != "" {
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", c.Port), cert, key, http.HandlerFunc(proxyHandler))
	} else {
		return http.ListenAndServe(fmt.Sprintf(":%d", c.Port), http.HandlerFunc(proxyHandler))
	}
}

func findDestination(routes []Route, source string) (string, error) {
	for _, route := range routes {
		if !route.Active {
			continue
		}
		if route.Source == source {
			return route.Dest, nil
		}
	}
	return "", errors.New("route not found")
}

func loadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	config.File = filepath

	return &config, nil
}

func (c *Config) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
