package line_utils

import (
	"bikefest/pkg/model"
	"encoding/json"
	"github.com/line/line-bot-sdk-go/linebot"
	"log"
)

func CreateFlexMessage(event *model.EventDetails) (*linebot.FlexContainer, error) {
	// Construct the Flex Message payload using event details
	contents := map[string]interface{}{
		"type": "bubble",
		"body": map[string]interface{}{
			"type":     "box",
			"layout":   "vertical",
			"contents": buildContents(event),
		},
	}

	contentsJSON, err := json.Marshal(contents)
	if err != nil {
		return nil, err
	}

	flexContainer, err := linebot.UnmarshalFlexMessageJSON(contentsJSON)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &flexContainer, nil
}

func buildContents(event *model.EventDetails) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type":   "text",
			"text":   event.Name,
			"weight": "bold",
			"size":   "xl",
			"wrap":   true,
		},
		{
			"type":   "text",
			"text":   event.Activity + " - " + event.Project,
			"wrap":   true,
			"margin": "md",
			"size":   "md",
		},
		{
			"type":   "text",
			"text":   event.Description,
			"wrap":   true,
			"margin": "md",
		},
		{
			"type":   "box",
			"layout": "baseline",
			"margin": "md",
			"contents": []map[string]interface{}{
				{
					"type":  "text",
					"text":  "日期:",
					"color": "#aaaaaa",
					"size":  "sm",
					"flex":  1,
				},
				{
					"type":  "text",
					"text":  event.Date,
					"wrap":  true,
					"color": "#666666",
					"size":  "sm",
					"flex":  5,
				},
			},
		},
		{
			"type":   "box",
			"layout": "baseline",
			"margin": "md",
			"contents": []map[string]interface{}{
				{
					"type":  "text",
					"text":  "時間:",
					"color": "#aaaaaa",
					"size":  "sm",
					"flex":  1,
				},
				{
					"type":  "text",
					"text":  event.StartTime + " - " + event.EndTime,
					"wrap":  true,
					"color": "#666666",
					"size":  "sm",
					"flex":  5,
				},
			},
		},
		{
			"type":   "box",
			"layout": "baseline",
			"margin": "md",
			"contents": []map[string]interface{}{
				{
					"type":  "text",
					"text":  "地點:",
					"color": "#aaaaaa",
					"size":  "sm",
					"flex":  1,
				},
				{
					"type":  "text",
					"text":  event.Location,
					"wrap":  true,
					"color": "#666666",
					"size":  "sm",
					"flex":  5,
				},
			},
		},
		{
			"type":   "box",
			"layout": "vertical",
			"margin": "md",
			"contents": []map[string]interface{}{
				{
					"type":   "button",
					"style":  "link",
					"height": "sm",
					"action": map[string]interface{}{
						"type":  "uri",
						"label": "報名參加",
						"uri":   event.Link,
					},
				},
			},
		},
	}
}
