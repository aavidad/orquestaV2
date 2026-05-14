package orquestaopesconnector

import (
	"context"
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
	var response []ExternalJobV0
	if err := client.getJSONV0(ctx, opesListJobsPathV0(query), &response); err != nil {
		return nil, err
	}
	return response, nil
}

func opesListJobsPathV0(query ExternalJobQueryV0) string {
	values := url.Values{}
	addQueryValueV0(values, "execution_mode", query.ExecutionMode)
	addQueryValueV0(values, "status", query.Status)
	addQueryValueV0(values, "job_type", query.JobType)
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
