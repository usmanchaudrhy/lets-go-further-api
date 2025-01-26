package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"greenlight.usman.com/internal/validator"
)

type envelop map[string]any

func (app *application) readIDParam(r *http.Request) (int64, error) {
	// When httprouter is parsing a request, any interpolated URL parameters will be stored
	// in the request context. We can use the ParamsFromContext() function to retrieve a slice
	// containing these parameter names and values
	params := httprouter.ParamsFromContext(r.Context())

	// We can use the ByName() method to get the value of the "ID" parameter from the slice
	// In our project all movies will have a unique positive integer ID, but the value returned
	// by ByName() is always a string. So we try to convert it to a base 10 integer (with a bit size of 64)
	// If the parameter could not be converted or is less then 1 we know the ID in invalid.
	// So we use http.NotFound() function to return a 404
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {

	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')
	// Its okay if the provided headers map is nil, Go does not throw an error
	// if you try to range over a nil map
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// Function responsible for reading the JSON body into a destination variable
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {

	// Usin the http.MaxBytesReader() to limit the size of the request ody to 1MB
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialise the json decoder and call the DisalloUnkownFields method on it
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// This is moved to above process
	// err := json.NewDecoder(r.Body).Decode(dst)
	err := dec.Decode(dst)

	if err != nil {
		// If there is an error while decoding the body, start the triage
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// we can use the errors.As() function to check the type of the error
		// syntaxError returns a plain english error message which includes the location of the error
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains bdaly-formed JSON (at character %d)", syntaxError.Offset)
		// In some cases Decode() my also return an io.ErrUnexpectedEOF error for syntax errors in JSON
		// We can check for this error using errors.Is and return a generix error message
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly formed-JSON")
		// This error occurs when JSON value is the wrong type for the target destination
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for the field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrext JSON at character %d", unmarshalTypeError.Offset)

		// An io.EOF error is returned if the body is empty
		// We can check for this using errors.Is
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		// If our JSON contains a field which cannotbe mapped to the target destination
		// then Decode() will now return an error message in the format "json: unkown field name"
		// We extract the error name from the error, and interpolate it into our own custom error message
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unkown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		// Check if the error has the type maxBytesError
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		default:
			return err
		}
	}

	// We can call decode again using a pointer to empty annonymous struct as the destination
	// If the request body cotnains a single JSON this returns an io.EOF error. So if we get anything else,
	// we know that there is additional data in the request body and we retutn our own custom error message
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// The readString() helper method returns a string value from the query string
// or the provided default value if no matching key could be found
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string
	// If no value is available will return empty string

	s := qs.Get(key)

	// if no key exists (or the value is empty)
	// then return the default value
	if s == "" {
		return defaultValue
	}

	// Otherwise return the string
	return s
}

// The readCSV() helper function reads a string value from the
// query string and then splits it into a slice on the comma character
// If no matching value is found it returns the default value
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	// returns the slice splitted on the comma
	return strings.Split(csv, ",")
}

// The readInt() helper reads a string value from the query string and tries to convert it into an integer
// If no matching value could be found, it returns the default value
// If the value couldn't be converted to an integer, then we record an error message in the provided validator instance
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	// Try to convert the value into an int, if the conversion fails all the value to the validator instance and return the value
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

// background() helper runs the function in the background goroutine
// handles all the errors and panic
func (app *application) background(fn func()) {
	// Launching a background goroutine
	app.wg.Add(1)
	go func() {

		defer app.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%v", err))
			}
		}()

		fn()
	}()
}
