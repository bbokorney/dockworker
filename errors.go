package main

import "fmt"

var (
	// ErrJobNotFound indicates the specificed job
	// could not be found
	ErrJobNotFound = fmt.Errorf("Job not found")
)

func errorResponse(msg string) errorMessage {
	return errorMessage{
		Message: msg,
	}
}

type errorMessage struct {
	Message string `json:"message"`
}
