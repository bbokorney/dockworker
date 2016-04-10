package main

import "fmt"

var (
	// ErrJobNotFound indicates the specificed job
	// could not be found
	ErrJobNotFound = fmt.Errorf("No job with that ID")

	// ErrInvalidJobID indicates the job ID was
	// specificed in an invalid format
	ErrInvalidJobID = fmt.Errorf("Invalid job ID")
)

func errorResponse(msg string) errorMessage {
	return errorMessage{
		Message: msg,
	}
}

type errorMessage struct {
	Message string `json:"message"`
}
