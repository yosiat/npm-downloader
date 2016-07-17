package models

import (
	"bytes"
	"crypto/sha1"
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
	Shasum       string `json:shasum`
	Tarball      string `json:tarball`
	NoAttachment bool   `json:noattachment`
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

// VersionsKeys - list of versions
func (p *Package) VersionsKeys() []string {
	keys := []string{}
	for k := range p.Versions {
		keys = append(keys, k)
	}

	return keys
}

// Download - download the given version to it's directory
func (v *Version) Download(downloadDirectory string) error {
	// Download the version
	response, err := http.Get(v.Dist.Tarball)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return fmt.Errorf("Failed to download %s - status code: %v", v.Dist.Tarball, response.StatusCode)
	}

	var responseBuffer bytes.Buffer
	_, err = io.Copy(&responseBuffer, response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}

	// Validate the hash
	shasum := fmt.Sprintf("%x", sha1.Sum(responseBuffer.Bytes()))
	if shasum != v.Dist.Shasum {
		return fmt.Errorf("ShasumMismatch: Shasum=%v, Remote=%v", shasum, v.Dist.Shasum)
	}

	// Save to file
	fileName, err := v.Dist.Filename()
	if err != nil {
		return err
	}

	file, err := os.Create(path.Join(downloadDirectory, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	file.Write(responseBuffer.Bytes())

	return nil
}

// Download - download the package.json and it's versions to the download directory
func (p *Package) Download(downloadDirectory string, versionsToDownload []string) error {
	// TODO: Wrap in zip, or add prefix for name dirs

	// Create package for the directory, we are using some autoincrement number as prefix
	packageDirectory := path.Join(downloadDirectory, fmt.Sprintf("%v-%s", p.Revision, p.Id))
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

// DownloadVersions - download all the given versions to the package directory
func (p *Package) DownloadVersions(packageDirectory string, versionsToDownload []string) error {
	if len(versionsToDownload) == 0 {
		return nil
	}

	errc := make(chan error, len(versionsToDownload))
	var wg sync.WaitGroup

	for _, versionNumber := range versionsToDownload {
		version := p.Versions[versionNumber]
		if version.Dist.NoAttachment {
			continue
		}

		wg.Add(1)
		go func(version Version) {

			// TODO: handle the error in here
			downloadErr := version.Download(packageDirectory)
			if downloadErr != nil {
				errc <- fmt.Errorf("Failed to download for %s\n%v", version.Dist.Tarball, downloadErr)
			}

			wg.Done()
		}(version)
	}

	wg.Wait()
	close(errc)

	var errors []error
	for err := range errc {
		errors = append(errors, err)
	}

	if len(errors) == 0 {
		return nil
	}

	return ErrAggregated{Errors: errors}
}

// ErrAggregated - is an aggregation of errors
type ErrAggregated struct {
	Errors []error
}

func (e ErrAggregated) Error() string {
	var buffer bytes.Buffer

	buffer.WriteString("Multiple errors: \n")
	for _, err := range e.Errors {
		fmt.Fprintln(&buffer, err.Error())
	}
	return buffer.String()
}
