package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/yosiat/npm-downloader/models"
)

// NpmRepository - handling fetches from npm
type NpmRepository struct {
	baseURL string
}

// FetchPackage - fetches package by id from npm registry
func (repository *NpmRepository) FetchPackage(packageID string) (models.Package, error) {
	packageLogger := log.WithFields(log.Fields{"packageID": packageID})

	packageURL := fmt.Sprintf("%s/%s", repository.baseURL, packageID)

	response, err := http.Get(packageURL)
	if err != nil {
		packageLogger.Errorf("Failed to fetch package from %s, error: %v", packageURL, err)
		return models.Package{}, fmt.Errorf("Failed to fetch package %s:\n%v", packageID, err)
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return models.Package{}, fmt.Errorf("Failed to read package body %s:\n%v", packageID, err)
	}

	if response.StatusCode != http.StatusOK {
		packageLogger.Errorf("Failed to fetch package from %s, got not ok status code: %v, response: %s", packageURL, response.StatusCode, body)
		return models.Package{}, fmt.Errorf("Failed to fetch package %s:\n%v", packageID, err)
	}

	pkg := models.Package{
		Id:   packageID,
		Blob: body,
	}

	json.Unmarshal(body, &pkg)

	return pkg, nil
}
