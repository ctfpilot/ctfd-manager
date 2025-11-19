package main

import (
	"encoding/json"
)

type Challenge struct {
	Schema              string   `json:"$schema"`
	Enabled             bool     `json:"enabled"`
	Name                string   `json:"name"`
	Slug                string   `json:"slug"`
	Author              string   `json:"author,omitempty"`
	Prerequisites       []string `json:"prerequisites,omitempty"`
	Category            string   `json:"category"`
	Difficulty          string   `json:"difficulty"`
	Tags                []string `json:"tags,omitempty"`
	Type                string   `json:"type"`
	InstancedType       string   `json:"instanced_type,omitempty"`
	InstancedName       string   `json:"instanced_name,omitempty"`
	InstancedSubdomains []string `json:"instanced_subdomains,omitempty"`
	Connection          string   `json:"connection,omitempty"` // Maximum length 255 characters
	Flag                []struct {
		Flag          string `json:"flag"`
		CaseSensitive bool   `json:"case_sensitive"`
	} `json:"flag,omitempty"`
	Points              int    `json:"points"`
	Decay               int    `json:"decay,omitempty"` // Maximum length 255 characters
	MinPoints           int    `json:"min_points"`
	DescriptionLocation string `json:"description_location,omitempty"`
	DockerfileLocations []struct {
		Context    string `json:"context"`
		Location   string `json:"location"`
		Identifier any    `json:"identifier"`
	} `json:"dockerfile_locations,omitempty"`
}

type ChallengeConfig struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Repository  string    `json:"repository"`
	Challenge   Challenge `json:"challenge"`
	Description string    `json:"description"`
	GeneratedAt string    `json:"generated_at"`
}

func jsonFormatChallenge(challenge Challenge) string {
	json, err := json.MarshalIndent(challenge, "", "  ")
	if err != nil {
		return ""
	}
	return string(json)
}

func jsonFormatChallengeConfig(challengeConfig *ChallengeConfig) string {
	json, err := json.MarshalIndent(challengeConfig, "", "  ")
	if err != nil {
		return ""
	}
	return string(json)
}

func filesDir(challengeConfig *ChallengeConfig) string {
	return "k8s/files"
}

func filesDirPath(challengeConfig *ChallengeConfig) string {
	// Get the files directory path from the challenge config
	filesDir := filesDir(challengeConfig)
	return challengeConfig.Path + "/" + filesDir
}

func getCategoryName(challengeConfig *ChallengeConfig, mappingMap MappingMap) string {
	// Get the category from the challenge config
	category := challengeConfig.Challenge.Category
	if category == "" {
		return "Uncategorized"
	}

	// Get difficulty from the challenge config
	difficulty := challengeConfig.Challenge.Difficulty

	// Check if difficulty exists in difficulty-categories mapping
	if categoryName, exists := mappingMap.DifficultyMapping[difficulty]; exists {
		return categoryName
	}

	// If not found, return the category as is
	if categoryName, exists := mappingMap.Categories[category]; exists {
		return categoryName
	}

	// Default to the original category if not found in mapping
	return category
}

func getDifficultyName(challengeConfig *ChallengeConfig, mappingMap MappingMap) string {
	// Get the difficulty from the challenge config
	difficulty := challengeConfig.Challenge.Difficulty
	if difficulty == "" {
		return "Unknown Difficulty"
	}

	// Check if difficulty exists in mapping
	if difficultyName, exists := mappingMap.Difficulties[difficulty]; exists {
		return difficultyName
	}

	// Default to the original difficulty if not found in mapping
	return difficulty
}
