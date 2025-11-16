package main

import (
	"encoding/json"
	"errors"

	ctfd "github.com/ctfer-io/go-ctfd/api"
	corev1 "k8s.io/api/core/v1"
)

type Page struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Route        string `json:"route"`
	AuthRequired bool   `json:"auth_required"`
	Nonce        string `json:"nonce"` // Whether the page open in current tab or a new one
	Draft        bool   `json:"draft"`
	Format       string `json:"format"`  // The format of the page, e.g., "markdown", "html"
	Enabled      bool   `json:"enabled"` // Whether the page is enabled or not
	Content      string `json:"content"` // The content of the page, which can be HTML or Markdown
}

type PageConfig struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Repository  string `json:"repository"`
	Page        Page   `json:"page"`
	GeneratedAt string `json:"generated_at"`
}

func (p *Page) toJSON() ([]byte, error) {
	return json.Marshal(p)
}

func (pc *PageConfig) toJSON() ([]byte, error) {
	return json.Marshal(pc)
}

func pageFromJSON(data []byte) (*Page, error) {
	var p Page
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func pageConfigFromJSON(data []byte) (*PageConfig, error) {
	var pc PageConfig
	err := json.Unmarshal(data, &pc)
	if err != nil {
		return nil, err
	}
	return &pc, nil
}

func postPageParamsFromPage(p *Page) (*ctfd.PostPagesParams, error) {
	params := &ctfd.PostPagesParams{
		Title:        p.Title,
		Route:        p.Route,
		AuthRequired: p.AuthRequired,
		Nonce:        p.Nonce,
		Draft:        p.Draft,
		Format:       p.Format,
		Hidden:       !p.Enabled,
		Content:      p.Content,
	}
	return params, nil
}

func patchPageParamsFromPage(p *Page) (*ctfd.PatchPageParams, error) {
	params := &ctfd.PatchPageParams{
		Title:        p.Title,
		Route:        p.Route,
		AuthRequired: p.AuthRequired,
		Nonce:        p.Nonce,
		Draft:        p.Draft,
		Format:       p.Format,
		Hidden:       !p.Enabled,
		Content:      p.Content,
	}
	return params, nil
}

func extractPageConfigMap(configMap *corev1.ConfigMap) (*PageConfig, error) {
	if configMap == nil {
		return nil, errors.New("configMap is nil")
	}

	// List of elements to check
	requiredElements := []string{
		"slug",
		"name",
		"path",
		"repository",
		"page",
		"generated_at",
	}

	// Check if the configmap has the required elements
	for _, element := range requiredElements {
		if _, ok := configMap.Data[element]; !ok {
			return nil, errors.New("Configmap does not contain the required element: " + element)
		}
	}

	pageConfig := &PageConfig{
		Slug:        configMap.Data["slug"],
		Name:        configMap.Data["name"],
		Path:        configMap.Data["path"],
		Repository:  configMap.Data["repository"],
		Page:        Page{},
		GeneratedAt: configMap.Data["generated_at"],
	}

	// Unmarshal the page data into the Page struct
	err := json.Unmarshal([]byte(configMap.Data["page"]), &pageConfig.Page)
	if err != nil {
		return nil, err
	}

	pageConfig.Page.Content = configMap.Data["content"]

	return pageConfig, nil
}
