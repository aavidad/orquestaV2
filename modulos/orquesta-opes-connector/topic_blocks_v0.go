package orquestaopesconnector

import (
	"context"
	"net/url"
	"strings"
)

func (client RESTClientV0) ListTopicBlocksV0(
	ctx context.Context,
	topicID string,
) ([]TopicBlockV0, error) {
	topicID = strings.TrimSpace(topicID)
	if client.baseURL == "" {
		return nil, connectorErrorV0{code: ErrOPESBaseURLRequiredV0}
	}
	if topicID == "" {
		return []TopicBlockV0{}, nil
	}
	var response []TopicBlockV0
	if err := client.getJSONV0(ctx, "/api/topics/"+url.PathEscape(topicID)+"/blocks", &response); err != nil {
		return nil, err
	}
	return response, nil
}
