package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

func (client RESTClientV0) ListExternalJobsV0(
	ctx context.Context,
	query ExternalJobQueryV0,
) ([]ExternalJobV0, error) {
	if client.baseURL == "" {
		return nil, connectorErrorV0{code: ErrOPESBaseURLRequiredV0}
	}
	if strings.TrimSpace(query.JobRef) != "" {
		var response ExternalJobV0
		if err := client.getJSONV0(ctx, opesGetJobPathV0(query.JobRef), &response); err != nil {
			return nil, err
		}
		return filterExternalJobsV0([]ExternalJobV0{response}, query), nil
	}
	var response []ExternalJobV0
	if err := client.getJSONV0(ctx, opesListJobsPathV0(query), &response); err != nil {
		return nil, err
	}
	return filterExternalJobsV0(response, query), nil
}

func opesGetJobPathV0(jobRef string) string {
	return DefaultOPESCreateJobPathV0 + "/" + url.PathEscape(strings.TrimSpace(jobRef))
}

func opesListJobsPathV0(query ExternalJobQueryV0) string {
	values := url.Values{}
	addQueryValueV0(values, "execution_mode", query.ExecutionMode)
	addQueryValueV0(values, "status", query.Status)
	addQueryValueV0(values, "job_type", query.JobType)
	addQueryValueV0(values, "job_ref", query.JobRef)
	addQueryValueV0(values, "program_id", query.ProgramID)
	addQueryValueV0(values, "topic_id", query.TopicID)
	addQueryValueV0(values, "correlation_id", query.CorrelationID)
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	encoded := values.Encode()
	if encoded == "" {
		return DefaultOPESCreateJobPathV0
	}
	return DefaultOPESCreateJobPathV0 + "?" + encoded
}

func addQueryValueV0(values url.Values, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values.Set(key, value)
	}
}

func filterExternalJobsV0(
	jobs []ExternalJobV0,
	query ExternalJobQueryV0,
) []ExternalJobV0 {
	out := make([]ExternalJobV0, 0, len(jobs))
	for _, job := range jobs {
		if !externalJobMatchesQueryV0(job, query) {
			continue
		}
		out = append(out, job)
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	if out == nil {
		return []ExternalJobV0{}
	}
	return out
}

func externalJobMatchesQueryV0(job ExternalJobV0, query ExternalJobQueryV0) bool {
	if query.JobRef != "" && strings.TrimSpace(job.ID) != strings.TrimSpace(query.JobRef) {
		return false
	}
	if query.JobType != "" && strings.TrimSpace(job.Type) != strings.TrimSpace(query.JobType) {
		return false
	}
	if query.Status != "" && strings.TrimSpace(job.Status) != strings.TrimSpace(query.Status) {
		return false
	}
	if query.ExecutionMode != "" && strings.TrimSpace(job.ExecutionMode) != strings.TrimSpace(query.ExecutionMode) {
		return false
	}
	if query.ProgramID != "" && externalJobScopedStringV0(job, "program_id") != strings.TrimSpace(query.ProgramID) {
		return false
	}
	if query.TopicID != "" && externalJobScopedStringV0(job, "topic_id") != strings.TrimSpace(query.TopicID) {
		return false
	}
	if query.CorrelationID != "" && strings.TrimSpace(job.CorrelationID) != strings.TrimSpace(query.CorrelationID) {
		return false
	}
	return true
}

func externalJobScopedStringV0(job ExternalJobV0, key string) string {
	value := externalJobPayloadStringV0(job.PayloadJSON, key)
	if value != "" {
		return value
	}
	return strings.TrimSpace(job.ExternalRefs[strings.TrimSpace(key)])
}

func externalJobPayloadStringV0(raw string, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	switch value := payload[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(value, 'f', -1, 64))
	default:
		return ""
	}
}
