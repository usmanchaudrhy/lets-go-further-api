package main

import "net/http"

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

}
