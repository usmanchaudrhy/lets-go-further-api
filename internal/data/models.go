package data

import (
	"database/sql"
	"errors"
)

// Define a custom ErrRecordNotFound error. We will return this from our
// Get() method when looking up a movie that do not exist in our DB

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Create a models struct that wraps the MovieModel.
// We are going to keep adding to this like the UserModel and the PermissionsModel
type Models struct {
	Movies MovieModel
}

// New() is responsible for initializing all the models
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
