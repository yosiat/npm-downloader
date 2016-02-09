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

func (repository *NpmRepository) FetchPackage(packageId string) models.Package {
	packageUrl := fmt.Sprintf("%s/%s", repository.baseUrl, packageId)

	response, err := http.Get(packageUrl)
	if err != nil {
		fmt.Printf("PKG - %s - %v\n", packageId, err)
		fmt.Errorf("Failed to fetch package %s:\n%v", packageId, err)
		return models.Package{}
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Errorf("Failed to read package body %s:\n%v", packageId, err)
		return models.Package{}
	}

	pkg := models.Package{
		Id:   packageId,
		Blob: body,
	}

	json.Unmarshal(body, &pkg)

	return pkg
}
