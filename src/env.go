package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

func getPassword() string {
	// Load data from env
	password := strings.TrimSpace(os.Getenv("PASSWORD"))
	if password == "" {
		log.Fatal("PASSWORD environment variable is not set")
	}
	return password
}

func getVersion() string {
	// Load data from env
	version := strings.TrimSpace(os.Getenv("VERSION"))
	if version == "" {
		return "0.0.0"
	}
	return version
}

func getNamespace() string {
	// Load data from env
	namespace := strings.TrimSpace(os.Getenv("NAMESPACE"))
	if namespace == "" {
		return "default"
	}
	return namespace
}

func getGithubToken() string {
	// Load data from env
	github_token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if github_token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is not set")
	}
	return github_token
}

func getGithubUser() string {
	// Load data from env
	github_user := strings.TrimSpace(os.Getenv("GITHUB_USER"))
	if github_user == "" {
		log.Fatal("GITHUB_USER environment variable is not set")
	}
	return github_user
}

func getGithubRepo() string {
	// Load data from env
	github_repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	if github_repo == "" {
		log.Fatal("GITHUB_REPO environment variable is not set")
	}
	return github_repo
}

func getGithubBranch() string {
	// Load data from env
	github_branch := strings.TrimSpace(os.Getenv("GITHUB_BRANCH"))
	if github_branch == "" {
		return "main"
	}
	return github_branch
}

func getCTFdURL() string {
	// Load data from env
	ctfd_url := strings.TrimSpace(os.Getenv("CTFD_URL"))
	if ctfd_url == "" {
		return "http://localhost:8000"
	}
	return ctfd_url
}

func getInstancedChallengeType() string {
	// Load data from env
	instanced_challenge_type := strings.TrimSpace(os.Getenv("INSTANCED_CHALLENGE_TYPE"))
	if instanced_challenge_type == "" {
		return "kubectf"
	}
	return instanced_challenge_type
}

func getChallengeDefaultPoints() int {
	// Load data from env
	default_points := strings.TrimSpace(os.Getenv("CHALLENGE_DEFAULT_POINTS"))
	if default_points == "" {
		return 1000
	}

	// Convert to int
	default_points_int, err := strconv.Atoi(default_points)
	if err != nil {
		log.Fatal("CHALLENGE_DEFAULT_POINTS environment variable is not a valid integer")
	}

	// Check if the value is greater than 0
	if default_points_int <= 0 {
		log.Fatal("CHALLENGE_DEFAULT_POINTS environment variable must be greater than 0")
	}

	return default_points_int
}

func getChallengeDefaultDecay() int {
	// Load data from env
	default_decay := strings.TrimSpace(os.Getenv("CHALLENGE_DEFAULT_DECAY"))
	if default_decay == "" {
		return 50
	}

	// Convert to int
	default_decay_int, err := strconv.Atoi(default_decay)
	if err != nil {
		log.Fatal("CHALLENGE_DEFAULT_DECAY environment variable is not a valid integer")
	}

	// Check if the value is greater than or equal to 0
	if default_decay_int < 0 {
		log.Fatal("CHALLENGE_DEFAULT_DECAY environment variable must be greater than or equal to 0")
	}

	return default_decay_int
}

func getChallengeDefaultMinPoints() int {
	// Load data from env
	default_min_points := strings.TrimSpace(os.Getenv("CHALLENGE_DEFAULT_MIN_POINTS"))
	if default_min_points == "" {
		return 100
	}

	// Convert to int
	default_min_points_int, err := strconv.Atoi(default_min_points)
	if err != nil {
		log.Fatal("CHALLENGE_DEFAULT_MIN_POINTS environment variable is not a valid integer")
	}

	// Check if the value is greater than or equal to 0
	if default_min_points_int < 0 {
		log.Fatal("CHALLENGE_DEFAULT_MIN_POINTS environment variable must be greater than or equal to 0")
	}

	return default_min_points_int
}
