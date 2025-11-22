package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// ---------
// Formats
// ---------

type ChallengesResponse struct {
    Challenges []map[string]string `json:"challenges"`
}

// ---------
// Challenges
// ---------

func middleware(w http.ResponseWriter, r *http.Request) error {
	// Check if the request is authenticated
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		errorResponse(w, r, http.StatusUnauthorized, "Unauthorized")
		return errors.New("Unauthorized")
	}

	// Strip the "Bearer " prefix if it exists
	if len(bearerToken) > 7 && bearerToken[:7] == "Bearer " {
		bearerToken = bearerToken[7:]
	}

	if !validatePassword(bearerToken) {
		errorResponse(w, r, http.StatusUnauthorized, "Unauthorized")
		return errors.New("Unauthorized")
	} else {
		log.Printf("Request authorized: %s %s", r.Method, r.URL.Path)
	}

	return nil
}

// ---------
// Challenges
// ---------

func getChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	// Get challenges from the cluster
	challenges, error := getConfigMapsByLabel(getNamespace(), map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

	if error != nil {
		log.Printf("Error getting challenges: %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting challenges")
		return
	}

	// Print challenges in json
	challengeList := make([]map[string]string, len(challenges))
	for i, challenge := range challenges {
		challengeList[i] = map[string]string{"name": challenge}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChallengesResponse{Challenges: challengeList})
}

func getChallengeHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	id := r.PathValue("id")

	if id == "" {
		errorResponse(w, r, http.StatusBadRequest, "Missing challenge ID")
		return
	}

	// Get challengeConfig from the cluster
	challengeConfig, error := getChallengeConfigMapByLabel(getNamespace(), id, map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

	if error != nil {
		log.Printf("Error getting challenge: %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "Challenge not found")
		return
	}

	// Print challenge in json
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"config\":%s}\n", jsonFormatChallengeConfig(challengeConfig))
}

func getChallengeFilesDirContentHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	id := r.PathValue("id")

	if id == "" {
		errorResponse(w, r, http.StatusBadRequest, "Missing challenge ID")
		return
	}

	// Get challengeConfig from the cluster
	challengeConfig, error := getChallengeConfigMapByLabel(getNamespace(), id, map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

	if error != nil {
		log.Printf("Error getting challenge: %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "Challenge not found")
		return
	}

	// Get the contents of the directory
	contents, error := getGithubDirContents(getGithubRepo(), getGithubBranch(), filesDirPath(challengeConfig))

	if error != nil {
		log.Printf("Error getting directory contents (err): %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting directory contents")
		return
	}
	if contents == nil {
		log.Printf("Error getting directory contents (contents): %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "Directory not found")
		return
	}

	// convert content to jsonData
	jsonData, error := json.MarshalIndent(contents, "", "  ")
	if error != nil {
		log.Printf("Error converting directory contents to json (err): %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error converting directory contents to json")
		return
	}

	if jsonData == nil {
		log.Printf("Error converting directory contents to json (json): %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error converting directory contents to json")
		return
	}

	// Print challenge in json
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"files\":%s}\n", jsonData)
}

func getChallengeFileHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	id := r.PathValue("id")
	file := r.PathValue("file")

	if id == "" || file == "" {
		errorResponse(w, r, http.StatusBadRequest, "Missing challenge ID or file name")
		return
	}

	// Get challengeConfig from the cluster
	challengeConfig, error := getChallengeConfigMapByLabel(getNamespace(), id, map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

	if error != nil {
		log.Printf("Error getting challenge: %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "Challenge not found")
		return
	}

	// Get all the files in the directory
	path := filesDirPath(challengeConfig)
	contents, error := getGithubDirContents(getGithubRepo(), getGithubBranch(), path)

	if error != nil {
		log.Printf("Error getting directory contents (err): %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting directory contents")
		return
	}
	if contents == nil {
		log.Printf("Error getting directory contents (contents): %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "Directory not found")
		return
	}

	// Check if the file exists in the directory
	fileExists := false
	for _, content := range contents {
		if content.GetName() == file {
			fileExists = true
			break
		}
	}
	if !fileExists {
		log.Printf("File not found in directory. Invalid file name provided")
		errorResponse(w, r, http.StatusNotFound, "File not found")
		return
	}

	// Get the contents of the file
	path = path + "/" + file
	data, error := getGithubFileBytes(getGithubRepo(), getGithubBranch(), path)

	if error != nil {
		log.Printf("Error getting file contents: %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting file contents")
		return
	}

	if data == nil {
		log.Printf("Error getting file contents (bytes): %s\n", error)
		errorResponse(w, r, http.StatusNotFound, "File not found")
		return
	}

	bytes := []byte(*data)

	// Set the content type based on the file extension
	contentType := http.DetectContentType(bytes)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+file)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(bytes)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(bytes); err != nil {
		log.Printf("Error writing file response: %s", err)
	}
	log.Printf("File %s sent successfully", file)
}

// ---------
// CTFd setup
// ---------

func postSetupHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	// Ensure post request
	if r.Method != http.MethodPost {
		errorResponse(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	postSetupCTFd(w, r)
}

func postUploadChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	// Get challenges from the cluster
	challenges, error := getConfigMapsByLabel(getNamespace(), map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

	if error != nil {
		log.Printf("Error getting challenges: %s\n", error)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting challenges")
		return
	}

	// Upload each challenge to CTFd
	for _, config := range challenges {
		challengeConfig, error := getChallengeConfigMapByLabel(getNamespace(), config, map[string]string{"challenges.kube-ctf.io/configmap": "challenge-config"})

		if error != nil {
			log.Printf("Error getting challenge: %s\n", error)
			errorResponse(w, r, http.StatusNotFound, "Challenge not found")
			return
		}

		id, err := updateOrCreateCTFdChallenge(challengeConfig)
		if err != nil {
			log.Printf("Error uploading challenge: %s\n", err)
			errorResponse(w, r, http.StatusInternalServerError, "Error uploading challenge")
			return
		}
		log.Printf("Uploaded challenge %d", id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"status\":\"ok\"}\n")
}

func getCTFdChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	// Get challenges
	challenges, err := getCTFdChallenges()
	if err != nil {
		log.Printf("Error getting challenges: %s\n", err)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting challenges")
		return
	}

	// Print challenges in json
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(challenges, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "{\"challenges\":[]}\n")
		return // Error converting challenges to json
	}

	fmt.Fprintf(w, "{\"challenges\":%s}\n", string(jsonData))
}

func getCTFdUploadedChallengesHandler(w http.ResponseWriter, r *http.Request) {
	// Authorize the request
	if err := middleware(w, r); err != nil {
		log.Printf("Middleware error: %s\n", err)
		return
	}

	// Get uploaded challenges
	uploadedChallenges, err := getUploadedCTFdChallenges()
	if err != nil {
		log.Printf("Error getting uploaded challenges: %s\n", err)
		errorResponse(w, r, http.StatusInternalServerError, "Error getting uploaded challenges")
		return
	}

	// Print uploaded challenges in json
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(uploadedChallenges, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "{\"uploaded_challenges\":[]}\n")
		return // Error converting challenges to json
	}

	fmt.Fprintf(w, "{\"uploaded_challenges\":%s}\n", string(jsonData))
}
