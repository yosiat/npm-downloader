package main

import (
	"fmt"

	"github.com/yosiat/npm-downloader/models"
)

const baseDir = "/Users/yosi/code/go/src/github.com/yosiat/npm-downloader"
const downloadDirectory = "/Volumes/Data/npm"

type PackageStatus struct {
	Error        error
	IsDownloaded bool
	Package      models.Package
}

func FindPackagesToDownload(downloadedPackages, changes map[string]models.PackageCommit) map[string]models.PackageCommit {
	result := make(map[string]models.PackageCommit)

	for pkgId, change := range changes {
		downloadedPackage, exists := downloadedPackages[pkgId]
		if !exists {
			result[pkgId] = change
		}

		if exists && downloadedPackage.Revision != change.Revision {
			result[pkgId] = change
		}
	}

	return result
}

// TODO: why we can't use Repository interface in here
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
		fmt.Printf("Versions to download for react: %+v", versionsToDownload)

		downloadErr := pkg.Download(downloadDirectory, versionsToDownload)

		if downloadErr != nil {
			results <- PackageStatus{Error: downloadErr, IsDownloaded: true, Package: pkg}
		} else {
			results <- PackageStatus{Error: nil, IsDownloaded: true, Package: pkg}
		}

	}
}

// TODO: Check what happens if we have error..
func main() {
	fmt.Println("npm-downloader (v0.1)")
	skim := NpmRepository{baseUrl: "https://skimdb.npmjs.com/registry"}

	commitsRepo, err := CreateCommitsRepository(baseDir)
	if err != nil {
		panic(err)
	}
	defer commitsRepo.Close()

	fmt.Println("Reading changes from file _changes and from the db")
	changes := ReadChanges(baseDir)
	downloadedPackages := commitsRepo.AllSucessfullPackages()
	packagesToDownload := FindPackagesToDownload(downloadedPackages, changes)

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
			statusText = "Downloaded with error"
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
