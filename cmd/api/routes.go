package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialize a new httprouter router instance
	router := httprouter.New()

	// httprouter allows us to set our own custom handlers when we initialize the router
	// they must satisfy the http.Handler interface
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	// Register the relevant mthods, URL patterns and handler function for our endpoints
	// using the HandlerFunc() method. Note that http.MethodGet and http.MethodPost are constants
	// whcih equate to the strings GET and POST respectively
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.listMovieHandler)
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)

	// Adding a route for the PATCH and DELETE movie method
	// PATCH - is used for partial updates
	// PUT - is used for completely replacing the record
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.updateMovieHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.deleteMovieHandler)

	// We are going to wrap the router function with the recoverPanic middleware
	return app.recoverPanic(app.rateLimit(router))
}
