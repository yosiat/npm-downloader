package main

import (
	"bytes"
	"encoding/gob"
	"path"

	"github.com/boltdb/bolt"
	"github.com/yosiat/npm-downloader/models"
)

type CommitsRepository struct {
	Db *bolt.DB
}

func CreateCommitsRepository(baseDir string) (CommitsRepository, error) {
	db, err := bolt.Open(path.Join(baseDir, "status"), 0600, nil)
	if err != nil {
		return CommitsRepository{}, err
	}

	tx, err := db.Begin(true)
	if err != nil {
		db.Close()
		return CommitsRepository{}, err
	}
	defer tx.Rollback()

	_, err = tx.CreateBucketIfNotExists([]byte("Success"))
	if err != nil {
		db.Close()
		return CommitsRepository{}, err
	}

	_, err = tx.CreateBucketIfNotExists([]byte("Error"))
	if err != nil {
		db.Close()
		return CommitsRepository{}, err
	}

	if err := tx.Commit(); err != nil {
		db.Close()
		return CommitsRepository{}, err
	}

	return CommitsRepository{Db: db}, nil
}

// Sucess - add success entry to the db
func (repo *CommitsRepository) Sucess(pkg models.Package) {
	repo.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Success"))

		packageCommit := models.CreatePackageCommit(pkg)
		var pkgCommitBuffer bytes.Buffer
		enc := gob.NewEncoder(&pkgCommitBuffer)

		err := enc.Encode(packageCommit)
		if err != nil {
			return err
		}

		return b.Put([]byte(pkg.Id), pkgCommitBuffer.Bytes())

	})
}

func (repo *CommitsRepository) Error(pkgID string, err error) {
	repo.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Error"))
		return b.Put([]byte(pkgID), []byte(err.Error()))
	})
}

// ErrorsCount - returns how much errors we have
func (repo *CommitsRepository) ErrorsCount() int {

	errorsCount := 0
	repo.Db.View(func(tx *bolt.Tx) error {
		successBucket := tx.Bucket([]byte("Error"))

		successBucket.ForEach(func(_, _ []byte) error {
			errorsCount++
			return nil
		})

		return nil
	})

	return errorsCount
}

// AllSucessfullPackages - returns all succesful packages download
func (repo *CommitsRepository) AllSucessfullPackages() map[string]models.PackageCommit {
	packageCommitByID := make(map[string]models.PackageCommit)

	repo.Db.View(func(tx *bolt.Tx) error {
		successBucket := tx.Bucket([]byte("Success"))

		successBucket.ForEach(func(packageId, packageCommitBuffer []byte) error {
			var pkgCommitStatus models.PackageCommit

			decoder := gob.NewDecoder(bytes.NewBuffer(packageCommitBuffer))
			err := decoder.Decode(&pkgCommitStatus)
			if err != nil {
				return err
			}

			packageCommitByID[pkgCommitStatus.Id] = pkgCommitStatus
			return nil
		})

		return nil
	})

	return packageCommitByID
}

func (repo *CommitsRepository) Close() {
	repo.Db.Close()
}
