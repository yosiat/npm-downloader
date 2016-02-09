package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

type VersionDist struct {
	Shasum  string `json:shasum`
	Tarball string `json:tarball`
}

func (v *VersionDist) Filename() (string, error) {
	url, err := url.Parse(v.Tarball)
	if err != nil {
		return "", err
	}

	pathFragments := strings.Split(url.Path, "/")

	return pathFragments[len(pathFragments)-1], nil
}

type Version struct {
	Dist VersionDist `json:dist`
}

type Package struct {
	Id       string             `json:"_id"`
	Revision string             `json:"_rev"`
	Versions map[string]Version `json:versions`

	// Saving the blob, saving it later to file
	Blob []byte `json:omit`
}

func (p *Package) VersionsKeys() []string {
	keys := []string{}
	for k := range p.Versions {
		keys = append(keys, k)
	}

	return keys
}

func (v *Version) Download(downloadDirectory string) error {
	fileName, err := v.Dist.Filename()
	if err != nil {
		return err
	}

	file, err := os.Create(path.Join(downloadDirectory, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	response, err := http.Get(v.Dist.Tarball)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func (p *Package) Download(downloadDirectory string, versionsToDownload []string) error {
	// Create package for the directory
	packageDirectory := path.Join(downloadDirectory, p.Id)
	err := os.MkdirAll(packageDirectory, os.ModePerm)
	if err != nil {
		return err
	}

	// Save the package to package.json
	err = ioutil.WriteFile(path.Join(packageDirectory, "package.json"), p.Blob, 0644)
	if err != nil {
		return err
	}

	return p.DownloadVersions(packageDirectory, versionsToDownload)
}

func (p *Package) DownloadVersions(packageDirectory string, versionsToDownload []string) error {
	if len(versionsToDownload) == 0 {
		return nil
	}

	var wg sync.WaitGroup

	for _, versionNumber := range versionsToDownload {
		version := p.Versions[versionNumber]
		wg.Add(1)
		go func(version Version) {
			defer wg.Done()

			downloadErr := version.Download(packageDirectory)
			if downloadErr != nil {
				fmt.Errorf("Failed to download for %s\n%v", version.Dist.Tarball, downloadErr)
			}
		}(version)
	}

	wg.Wait()

	return nil
}
