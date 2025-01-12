package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	// declaring an http server using the same settings as in out main() function
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	// create a shutdown channel to receive any errors returned by the graceful Shutdown function
	shutdownError := make(chan error)

	// Start a background goroutine to listen for OS signals for graceful shutdowns
	go func() {
		// create a new channel which carries os.Signal values
		quit := make(chan os.Signal, 1)

		// use signal.Notify() to listen for SIGTERM and SIGINT signals
		// and relay them to the quit channel. Any other signals will not be caught
		// by signal.Notify() and will retain their default behaviour
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// read on the sig channel, this will block until a signal is received
		s := <-quit

		// log the message to say that the signal has been caught. Notice that we also call
		// the String() method on the signal to get the signal name and include it in the entry
		app.logger.Info("shutting down server", "signal", s.String())

		// create a context with a 30 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		defer cancel()

		// call shutdown on the server passing the context
		// shutdown will return nil if there was no error
		shutdownError <- srv.Shutdown(ctx)

		// Exit the application with a 0 status code
		// os.Exit(0)

	}()

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)

	// calling Shutdown() on the server will cause ListenAndServe() to immediately return an
	// http.ErrServerClosed error. So if we see this error, it is actually a good thing and an
	// indication that the graceful shutdown has started.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// otherwise we wait to reeive the shutdown value on the channel
	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", srv.Addr)

	return nil
}
