package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"greenlight.usman.com/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// We are going to use this generic function to validate the movie struct passed in the request
func ValidateMovie(v *validator.Validator, movie *Movie) {
	// Use the check method to execute our validation checks. This will add the provided key
	// and error messages to the errors map if the check does not evaluate to true.
	// For example - in the first check we check if the title is not equal to an empty string
	// in the second, we check if the length of title is less then or equal to 500 bytes
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be  more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1988")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain atleast 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	// we can use the unique helper to check all the genres are unqie
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// MovieModel struct type will encapsulate all the code for reading and writing movie data to and from DB
// It wraps a DB connection pool
type MovieModel struct {
	DB *sql.DB
}

// Insert is responsible for inserting a new record in the movie DB
func (m MovieModel) Insert(movie *Movie) error {

	// Define a query to insert a new record in the movies table
	// RETURNING is a postgres specific clause which can be used to return values from the
	// row inserted, updated or deleted
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`

	// args is a slice contaning the values of the placeholders
	// pq.Array() is an adapter function takes our []string slice and converts it to a pq.StringArray type
	// we can also use this with bool, byte, int32, int64, float32 and float64 array types
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	// create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// Get returns a specific record from the move DB
func (m MovieModel) Get(id int64) (*Movie, error) {

	// Postgres bigserial that we are using as movie ID starts auto-incrementing at 1 by default
	// we can assume there will be not value less than that.
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	// Define the SQL query for retrieving the movie data
	// pg_sleep(8) this can used to set the pg driver to sleep for 8 seconds
	query := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1
	`

	var movie Movie

	// Use the context.WithTimeout() function to craete a context.Context which carries a 3-second timeout deadline
	// Note we are using the empty context.Background() as the parent context
	// Timeout countdown begins from the moment the context is created. Any time spent creating the
	// context and calling other functions will count towards the timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// we also need to cancel the timeout before the function returns
	// this is necessary to release the associated resources, thereby preventing a memory leak
	// without this resources won't be released untill 3 seconds or the parent context cancels
	defer cancel()

	// Note: we need to scan the target for genres column using the adapter method pq.Array()
	// Update the QueryRow method to use the QueryRowContext method for handling timeouts
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	// If there was no movie found, Scan() will return an sql.ErrNoRows error.
	// we check for this error and return our custom ErrRecordFound error instead
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

// Update updates a specific record in the movies table
func (m MovieModel) Update(movie *Movie) error {

	query := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
	`

	// args slice to contain the values of the placeholder parameters
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	// Create a 3 second timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			{
				return ErrEditConflict
			}
		default:
			{
				return err
			}
		}
	}
	return nil
}

// Delete deletes a specific record from the movies table
func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `DELETE FROM movies where id = $1;`

	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Exec method returns an sql.Result object that contains information about how many rows were effected
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	// call the rowsAffected method to get the number of rows affected by the query
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// Add a GetAll function that returns all the movies based on the filter values provided
func (m *MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT count(*) over(), id, created_at, title, year, runtime, genres, version
        FROM movies
        WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
        AND (genres @> $2 OR $2 = '{}')     
        ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4
		`, filters.sortColumn(), filters.sortDirection())

	// Create a local context to timeout after if the query does not respond in time
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	// Create a new movies array to hold all the movies
	totalRecords := 0
	movies := []*Movie{}

	// Loop over the query result and scan the values in
	for rows.Next() {
		var movie Movie
		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		// If there is no error append this movie to the list
		movies = append(movies, &movie)
	}

	// Check if the rows returned any error
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// we can now generate the metadata
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}
