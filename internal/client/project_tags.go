package infisicalclient

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound = errors.New("Resource not found")
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

func (client Client) CreateProjectTag(request CreateProjectTagRequest) (CreateProjectTagResponse, error) {
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

func (client Client) UpdateProjectTag(request UpdateProjectTagRequest) (UpdateProjectTagResponse, error) {
	var body UpdateProjectTagResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/workspace/" + request.ProjectID + "/tags/" + request.TagID)

	if err != nil {
		return UpdateProjectTagResponse{}, fmt.Errorf("CallUpdateProjectTag: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectTagResponse{}, fmt.Errorf("CallUpdateProjectTag: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) GetProjectTagByID(request GetProjectTagByIDRequest) (GetProjectTagByIDResponse, error) {
	var body GetProjectTagByIDResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get("api/v1/workspace/" + request.ProjectID + "/tags/" + request.TagID)

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectTagByIDResponse{}, ErrNotFound
	}

	if err != nil {
		return GetProjectTagByIDResponse{}, fmt.Errorf("CallGetProjectTag: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectTagByIDResponse{}, fmt.Errorf("CallGetProjectTag: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) GetProjectTagBySlug(request GetProjectTagBySlugRequest) (GetProjectTagBySlugResponse, error) {
	var body GetProjectTagBySlugResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get("api/v1/workspace/" + request.ProjectID + "/tags/slug/" + request.TagSlug)

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectTagBySlugResponse{}, ErrNotFound
	}

	if err != nil {
		return GetProjectTagBySlugResponse{}, fmt.Errorf("CallGetProjectTagBySlug: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectTagBySlugResponse{}, fmt.Errorf("CallGetProjectTagBySlug: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) DeleteProjectTag(request DeleteProjectTagRequest) (DeleteProjectTagResponse, error) {
	var body DeleteProjectTagResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/workspace/" + request.ProjectID + "/tags/" + request.TagID)

	if err != nil {
		return DeleteProjectTagResponse{}, fmt.Errorf("CallDeleteProjectTag: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectTagResponse{}, fmt.Errorf("CallDeleteProjectTag: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
