package models

import "github.com/yosiat/npm-downloader/utils"

type PackageCommit struct {
	Id       string
	Revision string
	Versions []string
}

// TODO: test those functions.. for fun (and maybe profit!)
func (pkgCommit *PackageCommit) VersionsToDownload(pkg PackageCommit) []string {
	currentVersionKeys := utils.ToSet(pkg.Versions)
	dbVersionKeys := utils.ToSet(pkgCommit.Versions)

	return utils.ToStrings(currentVersionKeys.Difference(dbVersionKeys).ToSlice())
}

func (pkgCommit *PackageCommit) IsChanged(pkg PackageCommit) bool {
	if pkg.Revision != pkg.Revision {
		return true
	}

	if len(pkgCommit.VersionsToDownload(pkg)) > 0 {
		return true
	}

	return false
}

func Take(m map[string]PackageCommit, nth int) map[string]PackageCommit {
	result := make(map[string]PackageCommit)

	if nth == 0 {
		return result
	}

	count := 0
	for key, value := range m {
		if count > nth {
			break
		}

		result[key] = value
		count++
	}

	return result

}
