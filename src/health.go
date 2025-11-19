package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

var HEALTH atomic.Bool

func setUnhealthy() {
	HEALTH.Store(false)
	log.Println("Service marked as unhealthy")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Index page, that just lists the name of the service and status
	w.Header().Set("Content-Type", "application/json")

	healthcheck := "ok"
	if !healthy() {
		healthcheck = "error"
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "{\"name\":\"CTFd manager\",\"status\":\"%s\"}\n", healthcheck)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Print status in json
	w.Header().Set("Content-Type", "application/json")

	healthcheck := "ok"
	if !healthy() {
		healthcheck = "error"
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "{\"status\":\"%s\"}\n", healthcheck)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	// Print version in json
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"version\":\"%s\"}\n", getVersion())
}

func healthy() bool {
	// Check if the service is healthy
	if checkAccess() != nil {
		return false
	}

	_, err := getConfigMaps(getNamespace())
	if err != nil {
		log.Printf("Error getting configmaps: %s\n", err)
		return false
	}

	if !HEALTH.Load() {
		log.Println("Service deemed unhealthy, due to manual unhealthy call")
		return false
	}

	return true
}
