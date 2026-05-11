package orquestaappchangedirectorsource

type AppChangeDirectorSourceIssueV0 struct {
	Field string
}

func (issue AppChangeDirectorSourceIssueV0) Error() string {
	if issue.Field == "" {
		return "app_change_director_source"
	}
	return "app_change_director_source: " + issue.Field
}
