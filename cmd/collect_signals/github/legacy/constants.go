package legacy

import (
	"errors"
	"time"
)

const (
	SinceDuration time.Duration = time.Hour * 24 * 30
	IssueLookback time.Duration = time.Hour * 24 * 90 * 24

	MaxContributorLimit         = 5000
	MaxTopContributors          = 15
	TooManyContributorsOrgCount = 10

	TooManyCommentsFrequency = 2.0

	releasesPerPage = 100
)

var (
	TooManyResultsError = errors.New("too many results")
)
