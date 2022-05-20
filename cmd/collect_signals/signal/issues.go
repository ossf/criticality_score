package signal

type IssuesSet struct {
	UpdatedCount     Field[int]     `signal:"updated_issues_count,legacy"`
	ClosedCount      Field[int]     `signal:"closed_issues_count,legacy"`
	CommentFrequency Field[float64] `signal:"issue_comment_frequency,legacy"`
}

func (r *IssuesSet) Namespace() Namespace {
	return NamespaceIssues
}
