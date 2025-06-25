package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetProjectTags      = "CallGetProjectTags"
	operationCreateProjectTag    = "CallCreateProjectTag"
	operationUpdateProjectTag    = "CallUpdateProjectTag"
	operationGetProjectTagByID   = "CallGetProjectTagByID"
	operationGetProjectTagBySlug = "CallGetProjectTagBySlug"
	operationDeleteProjectTag    = "CallDeleteProjectTag"
)

func (client Client) GetProjectTags(request GetProjectTagsRequest) (GetProjectTagsResponse, error) {
	var body GetProjectTagsResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/workspace/" + request.ProjectID + "/tags")

	if err != nil {
		return GetProjectTagsResponse{}, errors.NewGenericRequestError(operationGetProjectTags, err)
	}

	if response.IsError() {
		return GetProjectTagsResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectTags, response, nil)
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
		return CreateProjectTagResponse{}, errors.NewGenericRequestError(operationCreateProjectTag, err)
	}

	if response.IsError() {
		return CreateProjectTagResponse{}, errors.NewAPIErrorWithResponse(operationCreateProjectTag, response, nil)
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
		return UpdateProjectTagResponse{}, errors.NewGenericRequestError(operationUpdateProjectTag, err)
	}

	if response.IsError() {
		return UpdateProjectTagResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectTag, response, nil)
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
		return GetProjectTagByIDResponse{}, errors.NewGenericRequestError(operationGetProjectTagByID, err)
	}

	if response.IsError() {
		return GetProjectTagByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectTagByID, response, nil)
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
		return GetProjectTagBySlugResponse{}, errors.NewGenericRequestError(operationGetProjectTagBySlug, err)
	}

	if response.IsError() {
		return GetProjectTagBySlugResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectTagBySlug, response, nil)
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
		return DeleteProjectTagResponse{}, errors.NewGenericRequestError(operationDeleteProjectTag, err)
	}

	if response.IsError() {
		return DeleteProjectTagResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectTag, response, nil)
	}

	return body, nil
}
