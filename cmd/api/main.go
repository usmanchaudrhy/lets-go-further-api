package main

// Current chapter 8 - Advanced CRUD Operations

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"greenlight.usman.com/internal/data"
)

// Declare a string containing the application version number.
// Later we will generate this automatically at build time
const version = "1.0.0"

// Define a struct to hold all the configurations for our application
// For now the only configuration setting is the port that we want the server to listen on
// And an environment variable to identify the environment Production Staging Development etc
// We will read these configurations from command line flags
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

// Define an application struct to hold the dependencies ffor our HTTP handlers, helpers
// and middleware. At this moment it contains a copy of config struct and logger, but will grow to include more
type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {
	var cfg config

	// Read value of the port and env command-line flags into the config struct.
	// we default the port number to be 4000 and the environment 'development' if no flags
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// The DSN flag is responsible for reading the config string to connect to the DB
	// TODO: storing the dsn as an OS environment variable, the book stores it as GREENLIGHT_DB_DSN
	// And then use os.Getenv("GREENLIGHT_DB_DSN") - Not doing now, will do in the future

	// flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://greenlight:pa55word@localhost/greenlight?sslmode=disable", "Postgres DSB DB")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:pass123@localhost/greenlight?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "Postgres max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "Postgres max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "Postgres max idle timeout")

	// Create command line flags to read the setting values into the config struct.
	// Notice that we use true as the default for the 'enabled' setting?
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	// Initialize a new structured logger, which writes log entries to std out
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Connect to the DB
	// We call the openDB function to connect to the DB and create a connection pool
	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	logger.Info("database connection pool established")

	// Declare instance of application struct with config and logger
	// Using the models as dependency on the app struct we can pass this to any handler in the code
	// and as we keep on adding more models they will all be accessible to the handlers
	// and it is also very informative eg to inser a movie app.models.Movies.Insert(...)
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	// Declare a new servemux and add a /v1/healthcheck route which dispatches requests to
	// the healthcheckHandler method
	// using the new routes function here
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/healthcheck", app.healthcheckHandler)

	// Declare an HTTP server which listens on the port provided in the config struct
	// uses the servemux we creted above as the handler, has some sensible timeout settings
	// and writes any log messages to the structured logger at Error level
	// srv := &http.Server{
	// 	Addr:         fmt.Sprintf(":%d", cfg.port),
	// 	Handler:      app.routes(),
	// 	IdleTimeout:  time.Minute,
	// 	ReadTimeout:  5 * time.Second,
	// 	WriteTimeout: 5 * time.Second,
	// 	ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	// }

	// logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)

	// err = srv.ListenAndServe()
	// logger.Error(err.Error())
	// os.Exit(1)

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of connections in the pool (idle + open)
	// Passing a value less than or equal to 0 means there is not limit
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set a maximum number of idle connections in the pool
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Setting a maximum duration for the idle connections
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// Create a context with a 5-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new database connection, passing in the context
	// we created above. If the connection could not be established successfully within the 5 second deadling,
	// then this will return an error. If we get this error, or any other, we close the connection and return error
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
