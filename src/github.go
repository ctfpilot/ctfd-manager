package main

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/google/go-github/v70/github"
)

var githubClient *github.Client

func initGithubClient() {
	log.Println("Initializing GitHub client...")

	// Initialize the GitHub client
	githubClient = github.NewClient(nil).WithAuthToken(getGithubToken())

	// Check access to the GitHub API
	_, _, err := githubClient.Users.Get(context.Background(), getGithubUser())
	if err != nil {
		log.Println("Error checking access to GitHub API:", err)
		return
	}
	log.Println("GitHub client initialized successfully")
}

func splitRepo(repo string) (string, string) {
	// Split the repository name into owner and repo
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func getGithubDirContents(repo, branch, path string) ([]*github.RepositoryContent, error) {
	owner, repo := splitRepo(repo)

	// Get the contents of the directory
	_, contents, _, err := githubClient.Repositories.GetContents(context.Background(), owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err != nil {
		log.Printf("Error getting directory contents: %s\n", err)
		return nil, err
	}

	return contents, nil
}

func getGithubFileBytes(repo, branch, path string) (*string, error) {
	owner, repo := splitRepo(repo)

	// Get the file contents metadata
	fileContent, _, _, err := githubClient.Repositories.GetContents(context.Background(), owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err != nil {
		log.Printf("Error getting file contents: %s\n", err)
		return nil, err
	}

	// If file is larger than 1 MB, use DownloadContents
	if fileContent.GetSize() > 1024*1024 {
		rc, _, err := githubClient.Repositories.DownloadContents(context.Background(), owner, repo, path, &github.RepositoryContentGetOptions{
			Ref: branch,
		})
		if err != nil {
			log.Printf("Error downloading large file content: %s\n", err)
			return nil, err
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			log.Printf("Error reading downloaded file content: %s\n", err)
			return nil, err
		}
		content := string(data)
		return &content, nil
	}

	// For small files, decode the file content as before
	decodedContent, err := fileContent.GetContent()
	if err != nil {
		log.Printf("Error decoding file content: %s\n", err)
		return nil, err
	}

	return &decodedContent, nil
}
