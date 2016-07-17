package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/yosiat/npm-downloader/models"
)

const baseDir = "/Users/yosi/code/go/src/github.com/yosiat/npm-downloader"
const downloadDirectory = "/Volumes/Data/npm"

type PackageStatus struct {
	Error        error
	IsDownloaded bool
	Package      models.Package
}

// findPackagesToDownload given a list of already downloaded packages and the changes feed
// we return a list of packages to download
func findPackagesToDownload(downloadedPackages, changes map[string]models.PackageCommit) []models.PackageCommit {
	var packagesToDownload []models.PackageCommit

	for pkgID, change := range changes {
		if pkgID != "react" {
			continue
		}

		var downloadedPackage models.PackageCommit
		var exists bool

		// If didn't already downloaded this package
		if downloadedPackage, exists = downloadedPackages[pkgID]; !exists {
			packagesToDownload = append(packagesToDownload, change)
			continue
		}

		// If there is change in revision
		if downloadedPackage.Revision != change.Revision {
			packagesToDownload = append(packagesToDownload, change)
		}
	}

	return packagesToDownload
}

func packageWorker(repository NpmRepository,
	downloadedPackages map[string]models.PackageCommit,
	jobs <-chan models.PackageCommit,
	results chan<- PackageStatus) {

	for item := range jobs {
		// Fetch the package from remote
		pkg, err := repository.FetchPackage(item.Id)
		if err != nil {
			results <- PackageStatus{Error: err, IsDownloaded: true, Package: models.Package{Id: item.Id, Revision: item.Revision}}
			return
		}

		downloadedInfo := downloadedPackages[item.Id]
		pkgCommitStatus := models.CreatePackageCommit(pkg)

		// Check there are changes..
		if !downloadedInfo.IsChanged(pkgCommitStatus) {
			results <- PackageStatus{Error: nil, IsDownloaded: false, Package: pkg}
		}

		// We have changes :)
		versionsToDownload := downloadedInfo.VersionsToDownload(pkgCommitStatus)
		downloadErr := pkg.Download(downloadDirectory, versionsToDownload)

		if downloadErr != nil {
			results <- PackageStatus{Error: downloadErr, IsDownloaded: true, Package: pkg}
		} else {
			results <- PackageStatus{Error: nil, IsDownloaded: true, Package: pkg}
		}

	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{})

	fmt.Println("npm-downloader (v0.1)")
	skim := NpmRepository{baseURL: "https://skimdb.npmjs.com/registry"}

	commitsRepo, err := CreateCommitsRepository(baseDir)
	if err != nil {
		panic(err)
	}
	defer commitsRepo.Close()

	fmt.Println("Reading changes from file _changes and from the db")
	changes := ReadChanges(baseDir)
	downloadedPackages := commitsRepo.AllSucessfullPackages()
	packagesToDownload := findPackagesToDownload(downloadedPackages, changes)

	workersCount := 6
	jobsCount := len(packagesToDownload)
	results := make(chan PackageStatus, jobsCount)
	jobs := make(chan models.PackageCommit, jobsCount)

	// Initialize wokrers
	fmt.Printf("Starting %v workers\n", workersCount)
	for w := 1; w <= workersCount; w++ {
		go packageWorker(skim, downloadedPackages, jobs, results)
	}

	// Submit jobs
	fmt.Printf("Submitting %v jobs\n", jobsCount)
	for _, change := range packagesToDownload {
		jobs <- change
	}

	close(jobs)

	fmt.Println("Waiting for results..")
	for a := 1; a <= jobsCount; a++ {
		status := <-results

		statusText := ""
		if status.Error == nil {
			statusText = "Downloaded successfuly"
		} else {
			statusText = fmt.Sprintf("Downloaded with error: %s", status.Error)
		}

		if !status.IsDownloaded {
			statusText = "no changes found"
		}

		fmt.Printf("[%v] %s %s\n", a, status.Package.Id, statusText)

		if status.Error == nil {
			commitsRepo.Sucess(status.Package)
		} else {
			commitsRepo.Error(status.Package.Id, status.Error)
		}

		fmt.Printf("[%v] Commited %s\n", a, status.Package.Id)
	}

	fmt.Println("FINISHED!")
}
