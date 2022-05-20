package signal

import "time"

type RepoSet struct {
	URL      Field[string]
	Language Field[string]
	License  Field[string]

	StarCount Field[int]
	CreatedAt Field[time.Time]
	UpdatedAt Field[time.Time]

	CreatedSince Field[int] `signal:"legacy"`
	UpdatedSince Field[int] `signal:"legacy"`

	ContributorCount Field[int] `signal:"legacy"`
	OrgCount         Field[int] `signal:"legacy"`

	CommitFrequency    Field[float64] `signal:"legacy"`
	RecentReleaseCount Field[int]     `signal:"legacy"`
}

func (r *RepoSet) Namespace() Namespace {
	return NamespaceRepo
}
