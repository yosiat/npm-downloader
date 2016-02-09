package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/yosiat/npm-downloader/models"
)

type Repository interface {
	FetchPackage(packageId string) models.Package
}

type NpmRepository struct {
	baseUrl string
}

func (repository *NpmRepository) FetchPackage(packageId string) (models.Package, error) {
	packageUrl := fmt.Sprintf("%s/%s", repository.baseUrl, packageId)

	response, err := http.Get(packageUrl)
	if err != nil {
		fmt.Printf("PKG - %s - %v\n", packageId, err)
		return models.Package{}, fmt.Errorf("Failed to fetch package %s:\n%v", packageId, err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return models.Package{}, fmt.Errorf("Failed to read package body %s:\n%v", packageId, err)
	}

	pkg := models.Package{
		Id:   packageId,
		Blob: body,
	}

	json.Unmarshal(body, &pkg)

	return pkg, nil
}
