package githubapi

import "time"

// DefaultGraphQLEndpoint is the default URL for the GitHub GraphQL API.
const DefaultGraphQLEndpoint = "https://api.github.com/graphql"

// GitTimestamp is an ISO-8601 encoded date for use with the GitHub GraphQL API.
// Unlike the DateTime type, GitTimestamp is not converted in UTC.

type GitTimestamp struct{ time.Time }
