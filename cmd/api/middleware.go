package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// If there was a panic, set a connection close header on the respone, this acts as a trigger
				// to automatically close the connection after the response has been sent
				w.Header().Set("Connection", "close")
				// The value returned by the recover() has the type any, so can use fmt.Errorf()
				// to normalize it into an error and call our serverErrorReponse method on it.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimitv1(next http.Handler) http.Handler {
	// Initialize a new rate limiter which allows an average of 2 requests per second
	// with a maximum of 4 requests in a single burst
	limiter := rate.NewLimiter(2, 4)

	// the function that we are returning is a closure
	// which closes over the limiter variable
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// call the limiter Allow to see if the request is allowed
		// if it is not allowed we will call the rateLimitExceeded helper

		// when we call the Allow method on the limiter, exactly one token
		// will be consumed from the bucket. If no tokens are left in
		// the bucket then Allow will return false

		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {

	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// declare a mutex and map to hold the clients IP address & rate limiters
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// launch a background go-routine which removes old entries from the
	// clients map once every minute
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from
			// happening while the cleanup is taking place
			mu.Lock()

			// Loop through all the clients. If they haven't been seen
			// within the last 3 minutes, delete the entries from the map
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract the IP address from the request
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// lock the mutex to prevent the code from being executed concurrently
		mu.Lock()

		// if the IP address already exists in the map
		// if it does not exist we initialize and create a new map of
		// the rate limiter
		if _, found := clients[ip]; !found {
			// create and add a new client struct to the map
			// if it does not exist

			clients[ip] = &client{
				limiter: rate.NewLimiter(
					rate.Limit(app.config.limiter.rps),
					app.config.limiter.burst,
				),
			}
		}

		// update the lastseen time for the client
		clients[ip].lastSeen = time.Now()

		// Call the allow method on the rate limiter for the current
		// IP address and see if the request is allowed or not
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)

	})
}
