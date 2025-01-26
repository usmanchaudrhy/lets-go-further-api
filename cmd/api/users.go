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

	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	// Old process see the helper function above, it has been moved there now
	// Launch a goroutine in the background to send the welcome email
	// go func() {

	// 	// run a deffered function which uses recover() to catch any panic, and log an error
	// 	// message instead of terminating the applciation
	// 	defer func() {
	// 		if err := recover(); err != nil {
	// 			app.logger.Error(fmt.Sprintf("%v", err))
	// 		}
	// 	}()

	// 	// call the send method on the Mailer, passing in the user's email address,
	// 	// name of the template file, and the User struct containing the new users data
	// 	err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
	// 	if err != nil {
	// 		app.logger.Error(err.Error())
	// 	}

	// }()
	// write a JSON response contaning the user data
	// along with a 202 Accepted Status code, showing that the request has been accepted for processing
	err = app.writeJSON(w, http.StatusAccepted, envelop{
		"user": user,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
