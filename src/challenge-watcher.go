package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var watchedConfigMaps = "challenges.kube-ctf.io/configmap"

func initBackgroundChallengeWatcher() error {
	log.Println("Initializing background challenge watcher...")

	// Check cluster access has been initialized
	if clientset == nil {
		return errors.New("Kubernetes client not initialized")
	}

	// Initialize the watcher for challenges
	for {
		watcher, err := clientset.CoreV1().ConfigMaps(getNamespace()).Watch(context.TODO(), metav1.ListOptions{
			LabelSelector: watchedConfigMaps,
		})
		if err != nil {
			panic("Unable to create watcher")
		}
		processEvent(watcher.ResultChan())
	}
}

func processEvent(eventChannel <-chan watch.Event) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added, watch.Modified:
				if updatedMap, ok := event.Object.(*corev1.ConfigMap); ok {
					if !hasBeenDeployed(updatedMap) {
						log.Printf("Challenge configmap added or updated: %s\n", updatedMap.Name)
						configMapType := getConfigMapType(updatedMap)
						if configMapType == "challenge" {
							log.Printf("Challenge configmap added or updated: %s\n", updatedMap.Name)

							challengeConfigMap, err := extractChallengeConfigMap(updatedMap)
							if err != nil {
								log.Printf("Error extracting challenge configmap: %v\n", err)
								continue
							}

							id, err := updateOrCreateCTFdChallenge(challengeConfigMap)
							if err != nil {
								log.Printf("Error updating or creating challenge in CTFd: %v\n", err)
								continue
							} else {
								log.Printf("Challenge updated or created with ID: %d\n", id)
							}
						} else if configMapType == "page" {
							log.Printf("Page configmap added or updated: %s\n", updatedMap.Name)

							pageConfigMap, err := extractPageConfigMap(updatedMap)
							if err != nil {
								log.Printf("Error extracting page configmap: %v\n", err)
								continue
							}
							id, err := uploadOrUpdateCTFdPage(&pageConfigMap.Page)
							if err != nil {
								log.Printf("Error updating or creating page in CTFd: %v\n", err)
								continue
							} else {
								log.Printf("Page updated or created with ID: %d\n", id)
							}
						} else {
							log.Printf("Unknown configmap type for %s, skipping\n", updatedMap.Name)
							continue
						}

						// Store the hash of the configmap to avoid re-deploying unchanged challenges that have already been successfully deployed
						newHash, err := getHashForConfigMap(updatedMap)
						if err != nil {
							log.Printf("Error generating hash for configmap %s: %v\n", updatedMap.Name, err)
							continue
						}
						err = storeConfigmapHash(getNamespace(), updatedMap.Name, newHash)
						if err != nil {
							log.Printf("Error storing hash for configmap %s: %v\n", updatedMap.Name, err)
							continue
						}
					} else {
						log.Printf("Challenge configmap %s has not changed since last deployment, skipping update\n", updatedMap.Name)
					}
				}
			case watch.Deleted:
				if deletedMap, ok := event.Object.(*corev1.ConfigMap); ok {
					log.Printf("Challenge configmap deleted: %s\n", deletedMap.Name)

					configMapType := getConfigMapType(deletedMap)
					if configMapType == "challenge" {
						challengeConfigMap, err := extractChallengeConfigMap(deletedMap)
						if err != nil {
							log.Printf("Error extracting challenge configmap: %v\n", err)
							continue
						}

						err = disableCTFdChallenge(challengeConfigMap)
						if err != nil {
							log.Printf("Error disabling challenge in CTFd: %v\n", err)
						}

						storeConfigmapHash(getNamespace(), deletedMap.Name, "") // Clear the stored hash for this configmap
					} else if configMapType == "page" {
						pageConfigMap, err := extractPageConfigMap(deletedMap)
						if err != nil {
							log.Printf("Error extracting page configmap: %v\n", err)
							continue
						}

						err = deleteCTFdPage(pageConfigMap.Slug)
						if err != nil {
							log.Printf("Error deleting page in CTFd: %v\n", err)
						}

						storeConfigmapHash(getNamespace(), deletedMap.Name, "") // Clear the stored hash for this configmap
					} else {
						log.Printf("Unknown configmap type for %s, skipping deletion\n", deletedMap.Name)
					}
				}
			default:
				// Do nothing
				log.Printf("Unhandled event type: %s for configmap: %s\n", event.Type, event.Object.(*corev1.ConfigMap).Name)
			}
		} else {
			// If eventChannel is closed, it means the server has closed the connection
			setUnhealthy()
			log.Println("Event channel closed, stopping watcher")
			return
		}
	}
}

func getConfigMapType(configMap *corev1.ConfigMap) string {
	if configMap == nil {
		return "unknown"
	}

	// Log labels for debugging
	if _, ok := configMap.Labels["challenges.kube-ctf.io/configmap"]; ok {
		// Print debugging information
		log.Printf("ConfigMap %s has challenges.kube-ctf.io/configmap label with value: %s\n", configMap.Name, configMap.Labels["challenges.kube-ctf.io/configmap"])
		if configMap.Labels["challenges.kube-ctf.io/configmap"] == "challenge-config" {
			return "challenge"
		} else if configMap.Labels["challenges.kube-ctf.io/configmap"] == "page-config" {
			return "page"
		}
	}
	return "unknown"
}

func hasBeenDeployed(configMap *corev1.ConfigMap) bool {
	newHash, err := getHashForConfigMap(configMap)
	if err != nil {
		log.Printf("Error generating hash for configmap %s: %v\n", configMap.Name, err)
		return false
	}

	oldHash, err := getConfigmapStoredHash(getNamespace(), configMap.Name)
	if err != nil {
		log.Printf("Error getting stored hash for configmap %s: %v\n", configMap.Name, err)
		return false
	}
	return newHash == oldHash
}

// getConfigmapHashSet fetches the 'challenge-configmap-hashset' ConfigMap and returns a map of configmap names to their stored hashes.
func getConfigmapHashSet(namespace string) (map[string]string, error) {
	cm, err := getConfigMap(namespace, "challenge-configmap-hashset")
	if err != nil {
		return nil, err
	}
	// Assume the Data field is map[string]string where key is configmap name, value is hash
	return cm.Data, nil
}

// getConfigmapStoredHash returns the stored hash for a configmap from the hashset configmap
func getConfigmapStoredHash(namespace, configmapName string) (string, error) {
	hashSet, err := getConfigmapHashSet(namespace)
	if err != nil {
		return "", err
	}
	hash, ok := hashSet[configmapName]
	if !ok {
		return "", nil // Not found
	}
	return hash, nil
}

func storeConfigmapHash(namespace, configmapName, hash string) error {
	// Use getConfigMap to fetch, and updateConfigMap to update
	configMap, err := getConfigmapHashSet(namespace)
	if err != nil {
		return err
	}

	data := configMap
	if data == nil {
		data = make(map[string]string)
	}
	data[configmapName] = hash

	err = updateConfigMap(namespace, "challenge-configmap-hashset", data)
	if err != nil {
		return err
	}

	log.Printf("Stored hash for configmap %s: %s\n", configmapName, hash)
	return nil
}

func getHashForConfigMap(configMap *corev1.ConfigMap) (string, error) {
	// Convert the configmap to a JSON string
	data, err := json.Marshal(configMap.Data)
	if err != nil {
		return "", err
	}

	// Generate a hash from the JSON string
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
