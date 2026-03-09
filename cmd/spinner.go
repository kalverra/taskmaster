package cmd

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// startSpinner creates and starts a terminal spinner with the given message.
// The caller must call Stop() on the returned spinner when the operation completes.
// Set sp.FinalMSG before stopping to replace the spinner line with a completion message.
func startSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " " + message
	s.Start()
	return s
}
