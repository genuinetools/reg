package trivy

import (
	"log"
	"time"
)

// Trivy defines the client for retrieving information from the clair API.
type Trivy struct {
	Location string
	Logf     LogfCallback
}

// LogfCallback is the callback for formatting logs.
type LogfCallback func(format string, args ...interface{})

// Quiet discards logs silently.
func Quiet(format string, args ...interface{}) {}

// Log passes log messages to the logging package.
func Log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Opt holds the options for a new clair client.
type Opt struct {
	Debug    bool
	Timeout  time.Duration
}

// New creates a new Trivy struct with the given URL and credentials.
func New(location string, opt Opt) (*Trivy, error) {
	// set the logging
	logf := Quiet
	if opt.Debug {
		logf = Log
	}

	client := &Trivy{
		Location: location,
		Logf:     logf,
	}

	return client, nil
}

// Close closes the gRPC connection
func (c *Trivy) Close() error {
	return nil
}
