package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	ctfd "github.com/ctfer-io/go-ctfd/api"
)

type KubeCTFPostChallengeParams struct {
	ctfd.PostChallengesParams

	TemplateName string `json:"template_name,omitempty"`
	InstanceType string `json:"instance_type,omitempty"`
}

type KubeCTFPatchChallengeParams struct {
	ctfd.PatchChallengeParams

	TemplateName string `json:"template_name,omitempty"`
	InstanceType string `json:"instance_type,omitempty"`
}

type CTFdStateParams struct {
	State string `json:"state"`
}

func getCTFdChallenges() ([]*ctfd.Challenge, error) {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		return nil, err
	}

	// Get challenges
	view := "admin"
	challenges, err := client.GetChallenges(&ctfd.GetChallengesParams{
		View: &view,
	})
	if err != nil {
		return nil, err
	}

	// Check if challenges are empty
	if challenges == nil || len(challenges) == 0 {
		return nil, nil
	}

	return challenges, nil
}

func getUploadedCTFdChallenges() (map[string]string, error) {
	// Get configmap for uploaded challenges
	configMap, err := getConfigMap(getNamespace(), CTFDCHALLENGESCONFIGMAP)
	if err != nil {
		log.Println("Error getting CTFd challenges configmap:", err)
		return nil, errors.New("error getting CTFd challenges configmap")
	}

	if configMap == nil {
		log.Println("CTFd challenges configmap is nil")
		return nil, errors.New("CTFd challenges configmap is nil")
	}

	if configMap.Data == nil {
		log.Println("CTFd challenges configmap data is nil")
		return make(map[string]string), nil
	}

	return configMap.Data, nil
}

func getUploadedCTFdChallenge(challengeName string) (string, error) {
	// Get uploaded challenges
	uploadedChallenges, err := getUploadedCTFdChallenges()
	if err != nil {
		log.Println("Error getting uploaded challenges:", err)
		return "", errors.New("error getting uploaded challenges")
	}

	// Check if challenge is uploaded
	if _, ok := uploadedChallenges[challengeName]; !ok {
		log.Printf("Challenge %s not found in uploaded challenges\n", challengeName)
		return "", errors.New("challenge not found in uploaded challenges")
	}

	// Check if 0 (deleted)
	if uploadedChallenges[challengeName] == "0" {
		return "0", nil
	}

	if uploadedChallenges[challengeName] == "" {
		return "0", nil
	}

	return uploadedChallenges[challengeName], nil
}

func setUploadedCTFdChallenge(challengeName string, challengeID int) error {
	// Get configmap for uploaded challenges
	data, err := getUploadedCTFdChallenges()
	if err != nil {
		log.Println("Error getting CTFd challenges configmap:", err)
		return errors.New("error getting CTFd challenges configmap")
	}

	// Set challenge ID in configmap
	data[challengeName] = strconv.Itoa(challengeID)

	// Update configmap
	err = updateConfigMap(getNamespace(), CTFDCHALLENGESCONFIGMAP, data)
	if err != nil {
		log.Println("Error updating CTFd challenges configmap:", err)
		return errors.New("error updating CTFd challenges configmap")
	}

	return nil
}

func deleteUploadedCTFdChallenge(challengeName string) error {
	// Get configmap for uploaded challenges
	data, err := getUploadedCTFdChallenges()
	if err != nil {
		log.Println("Error getting CTFd challenges configmap:", err)
		return errors.New("error getting CTFd challenges configmap")
	}

	// Set challenge ID in configmap
	data[challengeName] = "0"

	// Update configmap
	err = updateConfigMap(getNamespace(), CTFDCHALLENGESCONFIGMAP, data)
	if err != nil {
		log.Println("Error updating CTFd challenges configmap:", err)
		return errors.New("error updating CTFd challenges configmap")
	}

	return nil
}

func uploadCTFdChallengeFile(id int, challenge *ChallengeConfig, client *ctfd.Client) (int, error) {
	// Get files
	files, err := getGithubDirContents(getGithubRepo(), getGithubBranch(), filesDirPath(challenge))
	if err != nil {
		log.Printf("Error getting directory contents (err): %s\n", err)
		return 0, err
	}

	filesContent := make([]*ctfd.InputFile, 0)
	if files != nil && len(files) > 0 {
		for _, file := range files {
			if file.GetName() == ".gitignore" || file.GetName() == ".gitkeep" {
				continue
			}

			// Get file content
			path := (filesDirPath(challenge) + "/" + file.GetName())
			data, error := getGithubFileBytes(getGithubRepo(), getGithubBranch(), path)

			if error != nil {
				log.Printf("Error getting file content: %s\n", error)
			}

			// Convert to format
			if error == nil && data != nil && file.GetName() != "" {
				filesContent = append(filesContent, &ctfd.InputFile{
					Name:    file.GetName(),
					Content: []byte(*data),
				})
			}
		}
	}

	// Print files
	if len(filesContent) > 0 {
		for _, file := range filesContent {
			log.Printf("File: %s\n", file.Name)
			// log.Printf("Content: %s\n", string(file.Content))
		}
	}

	// Upload files
	if len(filesContent) != 0 {
		_, err = client.PostFiles(&ctfd.PostFilesParams{
			Files:     filesContent,
			Challenge: &id,
		})
		if err != nil {
			log.Printf("Error uploading files: %s\n", err)
			return 0, err
		}
	}

	return id, nil
}

func uploadCTFdChallenge(challenge *ChallengeConfig, client *ctfd.Client) (int, error) {
	state := "visible"
	if !challenge.Challenge.Enabled {
		state = "hidden"
	}

	challType := "dynamic"
	if challenge.Challenge.Type == "instanced" {
		challType = "kubectf"
	}

	// Cut the first two lines from the description
	description := challenge.Description
	if challenge.Description != "" {
		// Split the description into lines
		lines := strings.Split(challenge.Description, "\n")
		// Check if there are at least two lines
		if len(lines) > 2 {
			// Join the lines after the first two
			description = strings.Join(lines[2:], "\n")
		}
	}

	challengeMappingMap, err := getMappingMap(getNamespace())
	if err != nil {
		log.Println("Error getting mapping map:", err)
		return 0, errors.New("error getting mapping map")
	}

	params := ctfd.PostChallengesParams{
		Name:           challenge.Challenge.Name,
		Category:       getCategoryName(challenge, challengeMappingMap),
		Description:    description,
		Initial:        &challenge.Challenge.Points,
		Decay:          &challenge.Challenge.Decay,
		Minimum:        &challenge.Challenge.MinPoints,
		State:          state,
		Type:           challType,
		ConnectionInfo: &challenge.Challenge.Connection,
	}

	var uploadedChallenge *ctfd.Challenge

	// Upload challenge
	if challenge.Challenge.Type == "instanced" {
		kubectfSlug := challenge.Challenge.Slug
		if challenge.Challenge.InstancedName != "" && challenge.Challenge.InstancedName != challenge.Challenge.Slug {
			kubectfSlug = challenge.Challenge.InstancedName
		}
		instanceType := challenge.Challenge.InstancedType
		if instanceType == "" {
			instanceType = "none" // Use "none" if instanced type is empty
		}

		if len(challenge.Challenge.InstancedSubdomains) > 0 {
			if strings.Contains(challenge.Challenge.InstancedSubdomains[0], ":") {
				instanceType = strings.Join(challenge.Challenge.InstancedSubdomains, ",")
			} else {
				instanceType = instanceType + ":" + strings.Join(challenge.Challenge.InstancedSubdomains, ",")
			}
		}

		chall := &ctfd.Challenge{}
		if err := client.Post("/challenges", &KubeCTFPostChallengeParams{
			PostChallengesParams: params,
			TemplateName:         kubectfSlug,
			InstanceType:         instanceType,
		}, &chall); err != nil {
			return 0, err
		}
		uploadedChallenge = chall
	} else {
		ch, err := client.PostChallenges(&params)
		if err != nil {
			return 0, err
		}
		uploadedChallenge = ch
	}

	// Upload files
	_, err = uploadCTFdChallengeFile(uploadedChallenge.ID, challenge, client)
	if err != nil {
		log.Printf("Error uploading files: %s\n", err)
		return 0, err
	}

	// Upload flag
	for _, flag := range challenge.Challenge.Flag {
		data := ""
		if !flag.CaseSensitive {
			data = "case_insensitive" // Use case_insensitive if not case sensitive
		}

		_, err = client.PostFlags(&ctfd.PostFlagsParams{
			Challenge: uploadedChallenge.ID,
			Content:   flag.Flag,
			Type:      "static",
			Data:      data,
		})
		if err != nil {
			log.Printf("Error uploading flag: %s\n", err)
			return 0, err
		}
	}

	// Upload tags
	if challenge.Challenge.Tags != nil && len(challenge.Challenge.Tags) > 0 {
		for _, tag := range challenge.Challenge.Tags {
			if tag == "" {
				log.Println("Empty tag found, skipping upload")
				continue
			}

			_, err = client.PostTags(&ctfd.PostTagsParams{
				Challenge: uploadedChallenge.ID,
				Value:     tag,
			})
			if err != nil {
				log.Printf("Error uploading tag: %s\n", err)
				// Continue with the next tag even if one fails
				continue
			}
		}
	}

	// Add to uploaded challenges
	err = setUploadedCTFdChallenge(challenge.Challenge.Slug, uploadedChallenge.ID)
	if err != nil {
		log.Printf("Error setting uploaded challenge: %s\n", err)
		return 0, err
	}

	return uploadedChallenge.ID, nil
}

func updateCTFdChallenge(challenge *ChallengeConfig, client *ctfd.Client) (int, error) {
	// Get uploaded challenge ID
	uploadedChallengeID, err := getUploadedCTFdChallenge(challenge.Challenge.Slug)
	if err != nil {
		log.Printf("Error getting uploaded challenge: %s\n", err)
		return 0, err
	}

	challengeId, err := strconv.Atoi(uploadedChallengeID)
	if err != nil {
		log.Printf("Error converting challenge ID: %s\n", err)
		return 0, err
	}

	// Check if challenge is uploaded
	challenges, err := getCTFdChallenges()
	if err != nil || challenges == nil || len(challenges) == 0 {
		return uploadCTFdChallenge(challenge, client)
	}

	// Check if challenge is uploaded
	found := false
	for _, ch := range challenges {
		if ch.ID == challengeId {
			found = true
			break
		}
	}
	if !found {
		return uploadCTFdChallenge(challenge, client)
	}

	// Format state
	state := "visible"
	if !challenge.Challenge.Enabled {
		state = "hidden"
	}

	// Cut the first two lines from the description
	description := challenge.Description
	if challenge.Description != "" {
		// Split the description into lines
		lines := strings.Split(challenge.Description, "\n")
		// Check if there are at least two lines
		if len(lines) > 2 {
			// Join the lines after the first two
			description = strings.Join(lines[2:], "\n")
		}
	}

	challengeMappingMap, err := getMappingMap(getNamespace())
	if err != nil {
		log.Println("Error getting mapping map:", err)
		return 0, errors.New("error getting mapping map")
	}

	params := ctfd.PatchChallengeParams{
		Name:           challenge.Challenge.Name,
		Category:       getCategoryName(challenge, challengeMappingMap),
		Description:    description,
		Initial:        &challenge.Challenge.Points,
		Decay:          &challenge.Challenge.Decay,
		Minimum:        &challenge.Challenge.MinPoints,
		State:          state,
		ConnectionInfo: &challenge.Challenge.Connection,
	}

	var uploadedChallenge *ctfd.Challenge

	// Upload challenge
	if challenge.Challenge.Type == "instanced" {
		kubectfSlug := challenge.Challenge.Slug
		if challenge.Challenge.InstancedName != "" && challenge.Challenge.InstancedName != challenge.Challenge.Slug {
			kubectfSlug = challenge.Challenge.InstancedName
		}
		instanceType := challenge.Challenge.InstancedType
		if instanceType == "" {
			instanceType = "none" // Use "none" if instanced type is empty
		}

		if len(challenge.Challenge.InstancedSubdomains) > 0 {
			if strings.Contains(challenge.Challenge.InstancedSubdomains[0], ":") {
				instanceType = strings.Join(challenge.Challenge.InstancedSubdomains, ",")
			} else {
				instanceType = instanceType + ":" + strings.Join(challenge.Challenge.InstancedSubdomains, ",")
			}
		}

		chall := &ctfd.Challenge{}
		if err := client.Patch(fmt.Sprintf("/challenges/%s", uploadedChallengeID), &KubeCTFPatchChallengeParams{
			PatchChallengeParams: params,
			TemplateName:         kubectfSlug,
			InstanceType:         instanceType,
		}, &chall); err != nil {
			return 0, err
		}
		uploadedChallenge = chall
	} else {
		ch, err := client.PatchChallenge(challengeId, &params)
		if err != nil {
			return 0, err
		}
		uploadedChallenge = ch
	}

	// Get files from CTFd
	ctfdFiles, err := client.GetChallengeFiles(challengeId)
	if err != nil {
		log.Printf("Error getting challenge files: %s\n", err)
		return 0, err
	}
	if ctfdFiles != nil && len(ctfdFiles) > 0 {
		for _, file := range ctfdFiles {
			// Delete files
			err = client.DeleteFile(strconv.Itoa(file.ID))
			if err != nil {
				log.Printf("Error deleting file: %s\n", err)
				return 0, err
			}
		}
	}

	// Upload files
	_, err = uploadCTFdChallengeFile(challengeId, challenge, client)
	if err != nil {
		log.Printf("Error uploading files: %s\n", err)
		return 0, err
	}

	// Get flags from CTFd
	ctfdFlags, err := client.GetChallengeFlags(challengeId)
	if err != nil {
		log.Printf("Error getting challenge flags: %s\n", err)
		return 0, err
	}

	if ctfdFlags != nil && len(ctfdFlags) > 0 {
		for _, flag := range ctfdFlags {
			// Delete flags
			err = client.DeleteFlag(strconv.Itoa(flag.ID))
			if err != nil {
				log.Printf("Error deleting flag: %s\n", err)
				return 0, err
			}
		}
	}

	// Upload flag
	for _, flag := range challenge.Challenge.Flag {
		data := ""
		if !flag.CaseSensitive {
			data = "case_insensitive" // Use case_insensitive if not case sensitive
		}

		_, err = client.PostFlags(&ctfd.PostFlagsParams{
			Challenge: uploadedChallenge.ID,
			Content:   flag.Flag,
			Type:      "static",
			Data:      data,
		})
		if err != nil {
			log.Printf("Error uploading flag: %s\n", err)
			return 0, err
		}
	}

	// Remove existing tags
	tags, err := client.GetTags(&ctfd.GetTagsParams{
		ChallengeID: &challengeId,
	})
	if err != nil {
		log.Printf("Error getting challenge tags: %s\n", err)
		return 0, err
	}
	// Delete existing tags
	if tags != nil && len(tags) > 0 {
		for _, tag := range tags {
			err = client.DeleteTag(strconv.Itoa(tag.ID))
			if err != nil {
				log.Printf("Error deleting tag %s: %s\n", tag.Value, err)
				return 0, err
			}
			log.Printf("Deleted tag %s\n", tag.Value)
		}
	} else {
		log.Println("No tags found to delete")
	}

	// Upload tags
	if challenge.Challenge.Tags != nil && len(challenge.Challenge.Tags) > 0 {
		for _, tag := range challenge.Challenge.Tags {
			if tag == "" {
				log.Println("Empty tag found, skipping upload")
				continue
			}

			_, err = client.PostTags(&ctfd.PostTagsParams{
				Challenge: uploadedChallenge.ID,
				Value:     tag,
			})
			if err != nil {
				log.Printf("Error uploading tag: %s\n", err)
				// Continue with the next tag even if one fails
				continue
			}
		}
	}

	// Add to uploaded challenges
	err = setUploadedCTFdChallenge(challenge.Challenge.Slug, uploadedChallenge.ID)
	if err != nil {
		log.Printf("Error setting uploaded challenge: %s\n", err)
		return 0, err
	}

	return uploadedChallenge.ID, nil
}

func updateOrCreateCTFdChallenge(challenge *ChallengeConfig) (int, error) {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		log.Printf("Error getting CTFd client: %s\n", err)
		return 0, err
	}

	// Check if challenge is uploaded
	uploadedChallengeID, _ := getUploadedCTFdChallenge(challenge.Challenge.Slug)
	// Ignore error for getting uploaded challenge, as it might indicate that it is not uploaded yet

	id := 0

	if uploadedChallengeID == "" || uploadedChallengeID == "0" {
		id, err = uploadCTFdChallenge(challenge, client)
	} else {
		id, err = updateCTFdChallenge(challenge, client)
	}

	if err != nil {
		log.Printf("Error uploading challenge %s: %s\n", challenge.Challenge.Name, err)
		return 0, err
	}

	return id, nil
}

func uploadChallenge(challenge *ChallengeConfig) (*int, error) {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		log.Printf("Error getting CTFd client: %s\n", err)
		return nil, err
	}

	// Upload challenges
	log.Printf("Uploading challenge %s...\n", challenge.Challenge.Name)
	id, err := uploadCTFdChallenge(challenge, client)
	if err != nil {
		log.Printf("Error uploading challenge %s: %s\n", challenge.Challenge.Name, err)
		return nil, err
	}
	log.Printf("Uploaded challenge %s with ID %d\n", challenge.Challenge.Name, id)

	return &id, nil
}

func disableCTFdChallenge(challenge *ChallengeConfig) error {
	client, err := getCTFdClient()
	if err != nil {
		log.Printf("Error getting CTFd client: %s\n", err)
		return err
	}

	// Get uploaded challenge ID
	uploadedChallengeID, err := getUploadedCTFdChallenge(challenge.Challenge.Slug)
	if err != nil || uploadedChallengeID == "" || uploadedChallengeID == "0" {
		log.Printf("Challenge %s not found in uploaded challenges, nothing to disable (error: %s)\n", challenge.Challenge.Slug, err)
		return nil
	}

	// Convert uploadedChallengeID to int
	uploadedChallengeIDInt, err := strconv.Atoi(uploadedChallengeID)
	if err != nil {
		log.Printf("Error converting uploaded challenge ID %s to int: %s\n", uploadedChallengeID, err)
		return err
	}

	params := CTFdStateParams{
		State: "hidden", // Set state to hidden
	}

	log.Printf("Disabling challenge %s (%d) in CTFd...\n", challenge.Challenge.Slug, uploadedChallengeIDInt)
	updateChallenge := &ctfd.Challenge{}
	err = client.Patch(fmt.Sprintf("/challenges/%d", uploadedChallengeIDInt), &params, updateChallenge)
	if err != nil {
		log.Printf("Error disabling challenge in CTFd: %s\n", err)
		return err
	}
	log.Printf("Challenge %s disabled (hidden) in CTFd\n", challenge.Challenge.Slug)

	return nil
}
