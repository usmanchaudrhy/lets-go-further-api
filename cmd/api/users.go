package main

import (
	"errors"
	"net/http"

	"greenlight.usman.com/internal/data"
	"greenlight.usman.com/internal/validator"
)

// Get /v1/users
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {

	// create an annonymous struct to hold the expected data from the request body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// copy the data from the request body into a new User struct

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	//we can use the password Set() method to generate and store
	// the hashed and plaintext passwords.
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	// validate the user struct and return the error message
	// to the client if any of the checks fail
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// insert the user data into the database
	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// write a JSON response contaning the user data
	// along with a 201 created status code
	err = app.writeJSON(w, http.StatusCreated, envelop{
		"user": user,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
