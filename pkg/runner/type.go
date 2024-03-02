package runner

import (
	"dcard-backend-2024/pkg/model"
	"encoding/json"
)

type Request struct {
	RequestID string
}

type Response struct {
	RequestID string
}

type CreateAdRequest struct {
	Request
	*model.Ad
}

func (r *CreateAdRequest) ToMap() (map[string]interface{}, error) {
	jsonData, err := json.Marshal(r.Ad)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}
	// result["start_at"] = a.StartAt.Format(time.RFC3339)
	// result["end_at"] = a.EndAt.Format(time.RFC3339)
	return result, nil
}

func (r *CreateAdRequest) FromMap(m map[string]interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonData, r.Ad); err != nil {
		return err
	}
	// r.StartAt, _ = time.Parse(time.RFC3339, m["start_at"].(string))
	// r.EndAt, _ = time.Parse(time.RFC3339, m["end_at"].(string))
	return nil
}

type CreateAdResponse struct {
	Response
	AdID string
	Err  error
}

type GetAdRequest struct {
	Request
	*model.GetAdRequest
}

type GetAdResponse struct {
	Response
	Ads   []*model.Ad
	Total int
	Err   error
}
