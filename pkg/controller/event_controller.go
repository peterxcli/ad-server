package controller

import (
	"bikefest/pkg/model"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type EventController struct {
	eventService model.EventService
	asynqService model.AsynqNotificationService
}

func NewEventController(eventService model.EventService, asynqService model.AsynqNotificationService) *EventController {
	return &EventController{
		eventService: eventService,
		asynqService: asynqService,
	}
}

// GetEventByID godoc
// @Summary Get an event by ID
// @Description Retrieves an event by ID
// @Tags Event
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} model.EventResponse
// @Failure 404 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /events/{id} [get]
func (ctrl *EventController) GetEventByID(c *gin.Context) {
	id := c.Param("id")
	event, err := ctrl.eventService.FindByID(c, id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, model.Response{
			Msg: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.EventResponse{
		Data: event,
	})
}

// GetAllEvent godoc
// @Summary Get all events
// @Description Retrieves a list of all events with pagination
// @Tags Event
// @Accept json
// @Produce json
// @Param page query int false "Page number for pagination"
// @Param limit query int false "Number of items per page for pagination"
// @Success 200 {object} model.EventListResponse "List of events"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /events [get]
func (ctrl *EventController) GetAllEvent(c *gin.Context) {
	page, limit := RetrievePagination(c)
	events, err := ctrl.eventService.FindAll(c, int64(page), int64(limit))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
	}

	c.JSON(http.StatusOK, model.EventListResponse{
		Data: events,
	})
}

// UpdateEvent godoc
// @Summary Update an event
// @Description Updates an event by ID with new details
// @Tags Event
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event ID"
// @Param event body model.CreateEventRequest true "Event Update Information"
// @Success 200 {object} model.EventResponse "Event successfully updated"
// @Failure 400 {object} model.Response "Bad Request - Invalid input"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /events/{id} [put]
func (ctrl *EventController) UpdateEvent(c *gin.Context) {
	// TODO: only allow admin to update event

	id := c.Param("id")
	identity, _ := RetrieveIdentity(c, true)
	if identity.UserID != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
			Msg: "Permission denied",
		})
		return
	}
	// _ = identity.UserID
	var request model.CreateEventRequest
	if err := c.ShouldBind(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: err.Error(),
		})
		return
	}
	event, err := ctrl.eventService.FindByID(c, id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	eventStartTimeP := parseEventTime(request.EventTimeStart, model.EventTimeLayout)
	eventEndTimeP := parseEventTime(request.EventTimeEnd, model.EventTimeLayout)
	updatedEvent := &model.Event{
		ID:             request.ID,
		EventTimeStart: eventStartTimeP,
		EventTimeEnd:   eventEndTimeP,
		EventDetail:    request.EventDetail,
	}
	_, err = ctrl.eventService.Update(c, updatedEvent)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.EventResponse{
		Data: event,
	})
}

// StoreAllEvent godoc
// @Summary Store all events from the json file in the frontend repo
// @Tags Event
// @Accept json
// @Produce json
// @Router /events/test-store-all [get]
func (ctrl *EventController) StoreAllEvent(c *gin.Context) {
	jsonURL := "https://raw.githubusercontent.com/gdsc-ncku/BikeFestival17th-Frontend/main/src/data/event.json"
	var eventDetails []model.EventDetails
	var events []*model.Event

	resp, err := http.Get(jsonURL)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&eventDetails); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	for _, eventDetail := range eventDetails {
		eventStartTimeP := parseEventTime("2024/"+eventDetail.Date+" "+eventDetail.StartTime, model.EventTimeLayout)
		eventEndTimeP := parseEventTime("2024/"+eventDetail.Date+" "+eventDetail.EndTime, model.EventTimeLayout)
		// Stringnify the event eventDetail
		eventDetailJson, err := json.Marshal(eventDetail)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
				Msg: err.Error(),
			})
			return
		}
		eventDetailStr := string(eventDetailJson)
		eventID := eventDetail.ID
		event := &model.Event{
			ID:             &eventID,
			EventTimeStart: eventStartTimeP,
			EventTimeEnd:   eventEndTimeP,
			EventDetail:    &eventDetailStr,
		}
		events = append(events, event)
	}
	err = ctrl.eventService.StoreAll(c, events)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Msg: "Store all events success",
	})
}

func parseEventTime(timeStr string, layout string) *time.Time {
	parts := strings.Split(timeStr, "/")
	dateParts := strings.Split(parts[2], " ")

	// Convert month and day parts to integers
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(dateParts[0])

	// Reassemble the string with leading zeros for month and day
	normalizedTimeString := fmt.Sprintf("%d/%02d/%02d %s", year, month, day, dateParts[1])
	t, err := time.Parse(layout, normalizedTimeString)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &t
}
