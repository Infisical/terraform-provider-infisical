package infisicalclient

import (
	"fmt"
)

func (client Client) GetProjectTags(request GetProjectTagsRequest) (GetProjectTagsResponse, error) {
	var body GetProjectTagsResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/workspace/" + request.ProjectID + "/tags")

	if err != nil {
		return GetProjectTagsResponse{}, fmt.Errorf("CallGetProjectTags: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectTagsResponse{}, fmt.Errorf("CallGetProjectTags: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body, nil
}

func (client Client) CreateProjectTags(request CreateProjectTagRequest) (CreateProjectTagResponse, error) {
	var body CreateProjectTagResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/workspace/" + request.ProjectID + "/tags")

	if err != nil {
		return CreateProjectTagResponse{}, fmt.Errorf("CallCreateProjectTag: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectTagResponse{}, fmt.Errorf("CallCreateProjectTag: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
