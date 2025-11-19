package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	ctfd "github.com/ctfer-io/go-ctfd/api"
)

func getCTFdPages() ([]*ctfd.Page, error) {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		return nil, err
	}

	pages, err := client.GetPages(&ctfd.GetPagesParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CTFd pages: %w", err)
	}
	return pages, nil
}

func getUploadedCTFdPages() (map[string]string, error) {
	// Get configmap for uploaded pages
	configMap, err := getConfigMap(getNamespace(), CTFDPAGESCONFIGMAP)
	if err != nil {
		log.Println("Error getting CTFd pages configmap:", err)
		return nil, errors.New("error getting CTFd pages configmap")
	}

	if configMap == nil {
		log.Println("CTFd pages configmap is nil")
		return nil, errors.New("CTFd pages configmap is nil")
	}

	if configMap.Data == nil {
		log.Println("CTFd pages configmap data is nil")
		return make(map[string]string), nil
	}

	return configMap.Data, nil
}

func getUploadedCTFdPage(slug string) (string, error) {
	// Get uploaded pages
	uploadedPages, err := getUploadedCTFdPages()
	if err != nil {
		log.Println("Error getting uploaded pages:", err)
		return "", errors.New("error getting uploaded pages")
	}

	// Check if page is uploaded
	if _, ok := uploadedPages[slug]; !ok {
		log.Printf("Page %s not found in uploaded pages\n", slug)
		return "0", errors.New("page not found in uploaded pages")
	}

	// Check if 0 (deleted)
	if uploadedPages[slug] == "0" {
		return "0", nil
	}

	if uploadedPages[slug] == "" {
		return "0", nil
	}

	return uploadedPages[slug], nil
}

func setUploadedCTFdPage(pageSlug string, pageID int) error {
	// Get configmap for uploaded pages
	data, err := getUploadedCTFdPages()
	if err != nil {
		log.Println("Error getting CTFd pages configmap:", err)
		return errors.New("error getting CTFd pages configmap")
	}

	// Set page ID in configmap
	data[pageSlug] = strconv.Itoa(pageID)

	// Update configmap
	err = updateConfigMap(getNamespace(), CTFDPAGESCONFIGMAP, data)
	if err != nil {
		log.Println("Error updating CTFd pages configmap:", err)
		return errors.New("error updating CTFd pages configmap")
	}

	return nil
}

func deleteUploadedCTFdPage(pageSlug string) error {
	// Get configmap for uploaded pages
	data, err := getUploadedCTFdPages()
	if err != nil {
		log.Println("Error getting CTFd pages configmap:", err)
		return errors.New("error getting CTFd pages configmap")
	}

	// Set page ID in configmap
	data[pageSlug] = "0"

	// Update configmap
	err = updateConfigMap(getNamespace(), CTFDPAGESCONFIGMAP, data)
	if err != nil {
		log.Println("Error updating CTFd pages configmap:", err)
		return errors.New("error updating CTFd pages configmap")
	}

	return nil
}

func uploadCTFdPage(page *Page, client *ctfd.Client) (int, error) {
	// Post page
	data, err := postPageParamsFromPage(page)
	if err != nil {
		return 0, fmt.Errorf("failed to convert page to post params: %w", err)
	}
	resp, err := client.PostPages(data)
	if err != nil {
		return 0, fmt.Errorf("failed to upload CTFd page: %w", err)
	}

	// Add to uploaded pages
	err = setUploadedCTFdPage(page.Slug, resp.ID)
	if err != nil {
		log.Printf("Error setting uploaded page: %s\n", err)
		return 0, err
	}

	log.Printf("Uploaded CTFd page %s with ID %d\n", page.Title, resp.ID)

	return resp.ID, nil
}

func updateCTFdPage(page *Page, client *ctfd.Client) (int, error) {
	// Get uploaded page ID
	uploadedPageID, err := getUploadedCTFdPage(page.Slug)
	if err != nil && uploadedPageID != "0" {
		log.Printf("Error getting uploaded page: %s\n", err)
		return 0, err
	}

	if uploadedPageID == "0" {
		// Page not uploaded, upload it
		return uploadCTFdPage(page, client)
	}

	// Patch page
	data, err := patchPageParamsFromPage(page)
	if err != nil {
		return 0, fmt.Errorf("failed to convert page to patch params: %w", err)
	}
	resp, err := client.PatchPage(uploadedPageID, data)
	if err != nil {
		return 0, fmt.Errorf("failed to update CTFd page: %w", err)
	}

	log.Printf("Updated CTFd page %s with ID %d\n", page.Title, resp.ID)

	return resp.ID, nil
}

func uploadOrUpdateCTFdPage(page *Page) (int, error) {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		return 0, fmt.Errorf("failed to get CTFd client: %w", err)
	}

	// Update or upload page
	pageID, err := updateCTFdPage(page, client)
	if err != nil {
		return 0, fmt.Errorf("failed to upload or update CTFd page: %w", err)
	}

	return pageID, nil
}

func disableCTFdPage(pageSlug string) error {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		return fmt.Errorf("failed to get CTFd client: %w", err)
	}

	// Get uploaded page ID
	uploadedPageID, err := getUploadedCTFdPage(pageSlug)
	if err != nil {
		log.Printf("Error getting uploaded page: %s\n", err)
		return err
	}

	if uploadedPageID == "0" {
		log.Printf("Page %s is already disabled\n", pageSlug)
		return nil // Already disabled
	}

	// Patch page to disable it
	data := &ctfd.PatchPageParams{Hidden: true}
	_, err = client.PatchPage(uploadedPageID, data)
	if err != nil {
		return fmt.Errorf("failed to disable CTFd page: %w", err)
	}

	log.Printf("Disabled CTFd page %s with ID %s\n", pageSlug, uploadedPageID)

	return deleteUploadedCTFdPage(pageSlug)
}

func deleteCTFdPage(pageSlug string) error {
	// Get client
	client, err := getCTFdClient()
	if err != nil {
		return fmt.Errorf("failed to get CTFd client: %w", err)
	}

	// Get uploaded page ID
	uploadedPageID, err := getUploadedCTFdPage(pageSlug)
	if err != nil {
		log.Printf("Error getting uploaded page: %s\n", err)
		return err
	}

	if uploadedPageID == "0" {
		log.Printf("Page %s is already deleted\n", pageSlug)
		return nil // Already deleted
	}

	// Delete page
	err = client.DeletePage(uploadedPageID)
	if err != nil {
		return fmt.Errorf("failed to delete CTFd page: %w", err)
	}

	log.Printf("Deleted CTFd page %s with ID %s\n", pageSlug, uploadedPageID)

	return deleteUploadedCTFdPage(pageSlug)
}
