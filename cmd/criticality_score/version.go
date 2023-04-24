package main

import "fmt"

// These variables are populated by GoReleaser. They can be set manually on the
// command line using -ldflags. For example:
//
//	go build -ldflags="-X 'main.version=x' -X 'main.date=y' -X 'main.commit=z'" ./cmd/criticality_score
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	if version == "dev" {
		fmt.Printf("dev build")
	} else {
		fmt.Printf("v%s (%s - %s)\n", version, date, commit)
	}
}
