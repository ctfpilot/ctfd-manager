package main

import (
	"log"
	"net/http"
)

func main() {
	HEALTH.Store(true)

	http.HandleFunc("/", indexHandler)

	http.HandleFunc("/api/challenges", getChallengesHandler)
	http.HandleFunc("/api/challenges/{id}", getChallengeHandler)
	http.HandleFunc("/api/challenges/{id}/files", getChallengeFilesDirContentHandler)
	http.HandleFunc("/api/challenges/{id}/files/{file}", getChallengeFileHandler)

	http.HandleFunc("/api/ctfd/setup", postSetupHandler)
	http.HandleFunc("/api/ctfd/challenges/init", postUploadChallengesHandler)
	http.HandleFunc("/api/ctfd/challenges", getCTFdChallengesHandler)
	http.HandleFunc("/api/ctfd/challenges/uploaded", getCTFdUploadedChallengesHandler)

	http.HandleFunc("/api/version", versionHandler)
	http.HandleFunc("/api/status", statusHandler)
	http.HandleFunc("/status", statusHandler)

	// Initialize modules
	initGithubClient()
	err := initClusterClient()
	if err != nil {
		log.Fatalf("Error initializing cluster client: %s", err)
	}

	go func() {
		err := initBackgroundChallengeWatcher()
		if err != nil {
			log.Printf("Error initializing background challenge watcher: %s", err)
			setUnhealthy()
		}
	}()

	log.Println("CTFd Manager started")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
