package line_utils

import (
	"encoding/json"
	"io"
	"net/http"
)

type FriendshipStatusResponse struct {
	FriendFlag bool `json:"friendFlag"`
}

func GetFriendshipStatus(accessToken string) (friendFlag bool, err error) {
	// Define the API endpoint
	url := "https://api.line.me/friendship/v1/status"

	// Create a new HTTP client
	client := &http.Client{}

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	// Add the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Unmarshal the JSON response
	var response FriendshipStatusResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return false, err
	}

	return response.FriendFlag, nil
}
