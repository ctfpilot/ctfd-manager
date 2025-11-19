package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var clientset *kubernetes.Clientset

func initClusterClient() error {
	log.Println("Initializing Kubernetes client...")

	// Create a new Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("Error creating in-cluster config:", err)
		return err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println("Error creating Kubernetes client:", err)
		return err
	}

	// Check access to the Kubernetes API
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		log.Println("Error checking access to Kubernetes API:", err)
		return err
	}

	// Set the clientset
	clientset = client

	log.Println("Kubernetes client initialized successfully")

	return nil
}

func checkAccess() error {
	// Check access to the Kubernetes API
	_, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	return nil
}

// Get configmaps in the given namespace
func getConfigMaps(namespace string) ([]string, error) {
	// Get the configmaps in the given namespace
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var configMapNames []string
	for _, configMap := range configMaps.Items {
		configMapNames = append(configMapNames, configMap.Name)
	}

	return configMapNames, nil
}

// Get configmaps based on a label selector in the given namespace
func getConfigMapsByLabel(namespace string, labelSelector map[string]string) ([]string, error) {
	// Get the configmaps in the given namespace with the label selector
	metaLabelSelector := metav1.LabelSelector{MatchLabels: labelSelector}
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(metaLabelSelector.MatchLabels).String(),
	})
	if err != nil {
		return nil, err
	}

	var configMapNames []string
	for _, configMap := range configMaps.Items {
		configMapNames = append(configMapNames, configMap.Name)
	}

	return configMapNames, nil
}

// Get configmap by name in the given namespace and return the configmap as a dictionary
func getConfigMap(namespace string, name string) (*corev1.ConfigMap, error) {
	// Get the configmap in the given namespace
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

// Get configmap by name and label in the given namespace and return a single configmap
func getChallengeConfigMapByLabel(namespace string, name string, labelSelector map[string]string) (*ChallengeConfig, error) {
	// Get all configmaps
	configmaps, err := getConfigMapsByLabel(namespace, labelSelector)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(configmaps, name) {
		return nil, errors.New("Configmap not found")
	}

	// Get the configmap in the given namespace
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if configMap == nil {
		return nil, errors.New("Configmap not found")
	}

	// Check if the configmap has the correct name
	if configMap.Name != name {
		return nil, errors.New("Configmap not found")
	}

	return extractChallengeConfigMap(configMap)
}

// Get configmap by name in the given namespace and return the config if it matches the Challenge struct
func getChallengeConfigMap(namespace string, name string) (*ChallengeConfig, error) {
	// Get the configmap in the given namespace
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return extractChallengeConfigMap(configMap)
}

func extractChallengeConfigMap(configMap *corev1.ConfigMap) (*ChallengeConfig, error) {
	// List of elements to check
	requiredElements := []string{
		"name",
		"path",
		"repository",
		"challenge",
		"description",
	}

	// Check if the configmap has the required elements
	for _, element := range requiredElements {
		if _, ok := configMap.Data[element]; !ok {
			return nil, errors.New("Configmap does not contain the required element: " + element)
		}
	}

	// Convert the configmap data to a ChallengeConfig struct
	challengeConfig := &ChallengeConfig{}

	challengeConfig.Name = configMap.Data["name"]
	challengeConfig.Path = configMap.Data["path"]
	challengeConfig.Repository = configMap.Data["repository"]
	challengeConfig.Description = configMap.Data["description"]
	challengeConfig.GeneratedAt = configMap.Data["generated_at"]
	challengeConfig.Challenge = Challenge{}
	err := json.Unmarshal([]byte(configMap.Data["challenge"]), &challengeConfig.Challenge)
	if err != nil {
		return nil, err
	}

	return challengeConfig, nil
}

// Update configmap in the given namespace with the given name and data
func updateConfigMap(namespace string, name string, data map[string]string) error {
	// Get the configmap in the given namespace
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if configMap == nil {
		return errors.New("Configmap not found")
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	// Update the configmap data
	for key, value := range data {
		configMap.Data[key] = value
	}

	// Update the configmap in the given namespace
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

type MappingMap struct {
	Categories        map[string]string `json:"categories"`
	Difficulties      map[string]string `json:"difficulties"`
	DifficultyMapping map[string]string `json:"difficulty-categories"`
}

// Get map that maps categories, difficulty levels to their respective names. Also includes difficulty-category mapping.
func getMappingMap(namespace string) (MappingMap, error) {
	// Get the configmap in the given namespace
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "mapping-map", metav1.GetOptions{})
	if err != nil {
		return MappingMap{}, err
	}

	if configMap == nil {
		log.Println("Configmap \"mapping-map\" not found. Defaulting to empty mapping map.")
		return MappingMap{}, nil
	}

	// Convert the configmap data to a MappingMap struct
	mappingMap := MappingMap{
		Categories:        make(map[string]string),
		Difficulties:      make(map[string]string),
		DifficultyMapping: make(map[string]string),
	}

	for key, value := range configMap.Data {
		switch {
		case key == "categories":
			if err := json.Unmarshal([]byte(value), &mappingMap.Categories); err != nil {
				return MappingMap{}, err
			}
		case key == "difficulties":
			if err := json.Unmarshal([]byte(value), &mappingMap.Difficulties); err != nil {
				return MappingMap{}, err
			}
		case key == "difficulty-categories":
			if err := json.Unmarshal([]byte(value), &mappingMap.DifficultyMapping); err != nil {
				return MappingMap{}, err
			}
		}
	}

	return mappingMap, nil
}
