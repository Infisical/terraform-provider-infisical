package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) GetGroupById(request GetGroupByIdRequest) (GetGroupByIdResponse, error) {
	var groupResponse GetGroupByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return GetGroupByIdResponse{}, ErrNotFound
	}

	if err != nil {
		return GetGroupByIdResponse{}, fmt.Errorf("CallGetGroupById: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetGroupByIdResponse{}, fmt.Errorf("CallGetGroupById: Unsuccessful response. [response=%s]", response)
	}

	return groupResponse, nil
}
