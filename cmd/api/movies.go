package main

import (
	"errors"
	"fmt"
	"net/http"

	"greenlight.usman.com/internal/data"
	"greenlight.usman.com/internal/validator"
)

// Data race condition = can occur when two or more goroutines try to use a piece of shared data
// at the same time, but the result of the operation is dependent on the exact order that the scheduler
// executes their instructions

// Solution - Optimistic Locking
// Optimistic locking is based on using version numbers, both records that are being updated have a version number
// and during the update if the version number in the DB is greater than the version no for the update the update is rejected

// Add a creteMovieHandler for the POST /v1./movies endpoint
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Declare an annonymous struct to hold the information we expect in HTTP body
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Because our validation now happens on the movie struct
	// we copy over the input struct into a movie struct and perform the valdiation
	// checks on the movie struct
	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	// Initialize a new valdiator instance
	v := validator.New()

	// we can use the Valid() method to see if any of the checks failed. If they did,
	// we can then use the failedValidationResponse helper to send a response to the client
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Call the Insert() method on our movies Model to create a record in the DB and update movie struct
	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// When sending the http response back we need to include the Location header to let the client know
	// wheich URL they can find the newly created resource at. We make an empty header map and then use the
	// Set() method to add a new Location header, interpolating the system generated ID for the new URL
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	// write a json response with a 201 status created
	err = app.writeJSON(w, http.StatusCreated, envelop{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Add a showMovieHandler for the GET /v1/movies:id endpoint
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// we call the Get() method to fetch the data for a specific movie
	// we also need to use the errors.Is() to check for ErrRecordNotFound

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Function responsible for handling the updates
func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the movie ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch the existing movie record from the database, sending a 404 if not exists
	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)

			return
		}
	}

	// Declare an input struct to hold the expected data from the client
	// We use pointers for the Title, Year and Runtime fields.
	// Reason - so that we can skip through the fields that are not passed by the user
	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  *[]string     `json:"genres"`
	}

	// read the request body struct into the input struct
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// copy values from the request body to the movie record

	// If the input.Title is nil we know that no corresponding "title" key/value pair was provided in the JSON
	// So we move on and leave the movie record as is. Otherwise we update the movie record with the new value
	if input.Title != nil {
		movie.Title = *input.Title // because title is a pointer now, we need to deference it
	}

	// Repeat for all types
	if input.Year != nil {
		movie.Year = *input.Year
	}

	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}

	if input.Genres != nil {
		movie.Genres = *input.Genres // Note that we don't need to deference a slice
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// pass the updated movie record to the new Update method
	// we also add the check to check for any edit conflict errors
	// if there are any edit conflicts we return the error
	err = app.models.Movies.Update(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// if the movie has been successfully updated, write the movie response in a JSON
	err = app.writeJSON(w, http.StatusOK, envelop{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Delete movie handler
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// delete the movie from the database
	// sending a 404 response if no matching record found
	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// return a 200 OK status code, along with a success message
	err = app.writeJSON(w, http.StatusOK, envelop{"message": "movie successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listMovieHandler(w http.ResponseWriter, r *http.Request) {
	// To keep things consistent with our other handlers, we'll define an input struct
	// to hold the expected values from the request query string
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()

	// Call the r.URL.Query function to the url.Values map containing the query string data
	qs := r.URL.Query()

	// Using the helpers to extract the title, genres query string values
	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})

	// Get the page and page_size query string values as integers, Notice that we set the default value of
	// the page to 1 and default of page_size to 20, and that we pass the validator isntance as the final argument here
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	// Extract the sort query string value, falling back to id if the value is not provided
	input.Filters.Sort = app.readString(qs, "sort", "id")
	// we are going to set the sorted safelist value
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-runtime"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// use the GetAll function in movies to get all the movies array
	movies, metadata, err := app.models.Movies.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelop{"movies": movies, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
