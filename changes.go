package main

import (
	"encoding/json"
	"os"
	"path"

	"github.com/yosiat/npm-downloader/models"
)

type Changes struct {
	Results []struct {
		Id      string `json:"id"`
		Deleted bool   `json:"deleted"`

		Changes []struct {
			Revision string `json:"rev"`
		} `json:changes`
	} `json:results`
}

func ReadChanges(baseDir string) map[string]models.PackageCommit {
	file, err := os.OpenFile(path.Join(baseDir, "_changes"), os.O_RDWR, 0660)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(file)

	changes := Changes{}
	decoder.Decode(&changes)

	packageCommitById := make(map[string]models.PackageCommit)
	for _, change := range changes.Results {
		if change.Deleted {
			continue
		}

		packageCommitById[change.Id] = models.PackageCommit{
			Id:       change.Id,
			Revision: change.Changes[0].Revision,
		}
	}

	return packageCommitById
}
