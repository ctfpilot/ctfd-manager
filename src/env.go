package main

import (
	"os"
	"strings"
	"log"
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
		return "github-token"
	}
	return github_token
}

func getGithubUser() string {
	// Load data from env
	github_user := strings.TrimSpace(os.Getenv("GITHUB_USER"))
	if github_user == "" {
		return "github-user"
	}
	return github_user
}

func getGithubRepo() string {
	// Load data from env
	github_repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	if github_repo == "" {
		return "github-repo"
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
