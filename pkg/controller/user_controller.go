package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bikefest/pkg/bootstrap"
	"bikefest/pkg/model"
	"bikefest/pkg/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userSvc      model.UserService
	eventSvc     model.EventService
	asynqService model.AsynqNotificationService
	env          *bootstrap.Env
}

func NewUserController(userSvc model.UserService, eventSvc model.EventService, asynqService model.AsynqNotificationService, env *bootstrap.Env) *UserController {
	return &UserController{
		userSvc:      userSvc,
		eventSvc:     eventSvc,
		asynqService: asynqService,
		env:          env,
	}
}

// Profile godoc
// @Summary Profile
// @Description Fetches the profile of a user
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} model.UserResponse "Profile successfully retrieved"
// @Failure 401 {object} model.Response "Unauthorized: Invalid or expired token"
// @Failure 404 {object} model.Response "User not found"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/profile [get]
func (ctrl *UserController) Profile(c *gin.Context) {
	identity, _ := RetrieveIdentity(c, true)
	userID := identity.UserID
	profile, err := ctrl.userSvc.GetUserByID(c, userID)
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, model.Response{
			Msg: "User not found",
		})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, model.UserResponse{
		Data: profile,
	})
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Retrieves a user's information by their ID
// @Tags User
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} model.UserResponse "User successfully retrieved"
// @Failure 404 {object} model.Response "User not found"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/{user_id} [get]
func (ctrl *UserController) GetUserByID(c *gin.Context) {
	userID := c.Param("user_id")
	user, err := ctrl.userSvc.GetUserByID(c, userID)
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, model.Response{
			Msg: "User not found",
		})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, model.UserResponse{
		Msg:  "Get user by id",
		Data: user,
	})
}

// RefreshToken godoc
// @Summary Refresh User Token
// @Description Refreshes the access and refresh tokens for a user
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} model.TokenResponse "Access and Refresh Tokens successfully generated"
// @Failure 400 {object} model.Response "Bad Request - Invalid request format"
// @Failure 401 {object} model.Response "Unauthorized - Invalid or expired refresh token"
// @Failure 500 {object} model.Response "Internal Server Error - Error generating tokens"
// @Router /users/refresh_token [get]
func (ctrl *UserController) RefreshToken(c *gin.Context) {
	// fetch refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
			Msg: "Old authorization not found",
		})
		return
	}

	identity, err := ctrl.userSvc.VerifyRefreshToken(c, refreshToken, ctrl.env.JWT.RefreshTokenSecret)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
			Msg: err.Error(),
		})
		return
	}

	accessToken, err := ctrl.userSvc.CreateAccessToken(c, identity, ctrl.env.JWT.AccessTokenSecret, ctrl.env.JWT.AccessTokenExpiry)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	refreshToken, err = ctrl.userSvc.CreateRefreshToken(c, identity, ctrl.env.JWT.RefreshTokenSecret, ctrl.env.JWT.RefreshTokenExpiry)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	loginResponse := &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.SetCookie("access_token", fmt.Sprintf("Bearer %s", accessToken), 3600, "/", "", false, true)
	c.SetCookie("refresh_token", fmt.Sprintf("Bearer %s", refreshToken), 3600, "/", "", false, true)

	c.JSON(http.StatusOK, model.TokenResponse{
		Data: loginResponse,
	})
}

// GetUsers godoc
// @Summary Get Users
// @Description Retrieves a list of users
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} model.UserListResponse "List of users successfully retrieved"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users [get]
func (ctrl *UserController) GetUsers(c *gin.Context) {
	// page, limit := RetrievePagination(c)
	users, err := ctrl.userSvc.ListUsers(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, model.UserListResponse{
		Msg:  "Get users",
		Data: users,
	})
}

// Logout godoc
// @Summary User logout
// @Description Logs out a user by invalidating their authentication token
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} model.Response "Logout successful"
// @Failure 401 {object} model.Response "Unauthorized: Invalid token format"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/logout [post]
func (ctrl *UserController) Logout(c *gin.Context) {
	// TODO: need to discuss where to read the token from (header or body or cookie)
	authCookie, err := c.Cookie("access_token")
	bearerToken := strings.Split(authCookie, " ")
	if len(bearerToken) != 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
			Msg: "Invalid token format (length different from 2)",
		})
		return
	}
	authToken := bearerToken[1]
	err = ctrl.userSvc.Logout(c, &authToken, ctrl.env.JWT.AccessTokenSecret)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, model.Response{
		Msg: "Logout success",
	})
}

// FakeLogin godoc
// @Summary Fake Login
// @Description Simulates a login process for a user by generating fake access and refresh tokens
// @Tags User
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} model.TokenResponse "Login successful, tokens generated"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/login/{user_id} [get]
func (ctrl *UserController) FakeLogin(c *gin.Context) {
	userID := c.Param("user_id")

	accessToken, err := ctrl.userSvc.CreateAccessToken(c, &model.User{ID: userID}, ctrl.env.JWT.AccessTokenSecret, ctrl.env.JWT.AccessTokenExpiry)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	refreshToken, err := ctrl.userSvc.CreateRefreshToken(c, &model.User{ID: userID}, ctrl.env.JWT.RefreshTokenSecret, ctrl.env.JWT.RefreshTokenExpiry)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	loginResponse := &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.JSON(http.StatusOK, model.TokenResponse{
		Data: loginResponse,
	})

	// set to cookie
	c.SetCookie("access_token", strconv.FormatInt(ctrl.env.JWT.AccessTokenExpiry, 10), 3600, "/", "", false, true)
	c.SetCookie("refresh_token", strconv.FormatInt(ctrl.env.JWT.AccessTokenExpiry, 10), 3600, "/", "", false, true)
}

// FakeRegister godoc
// @Summary Fake Register
// @Description Register a fake user for testing purposes
// @Tags User
// @Accept json
// @Produce json
// @Param request body model.CreateFakeUserRequest true "Create Fake User Request"
// @Success 200 {object} model.Response "Fake register successful"
// @Failure 400 {object} model.Response "Bad Request - Invalid input data"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/register [post]
func (ctrl *UserController) FakeRegister(c *gin.Context) {
	var request model.CreateFakeUserRequest

	if err := c.ShouldBind(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: err.Error(),
		})
		return
	}

	user := &model.User{
		Name: request.Name,
	}

	err := ctrl.userSvc.CreateFakeUser(c, user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Msg: "Fake register success",
	})
}

// SubscribeEvent godoc
// @Summary Subscribe to an event
// @Description Subscribes a user to an event with the provided details
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body model.CreateEventRequest true "Event Subscription Request"
// @Success 200 {object} model.EventResponse "Successfully subscribed to the event"
// @Failure 400 {object} model.Response "Bad Request - Invalid input, such as invalid time format or missing required fields"
// @Failure 409 {object} model.Response "Conflict - User already subscribed to the event"
// @Failure 422 {object} model.Response "Unprocessable Entity - User has exceeded the maximum number of subscriptions"
// @Failure 500 {object} model.Response "Internal Server Error - Error storing the event, subscribing the user, or enqueuing the event notification"
// @Router /users/events [post]
func (ctrl *UserController) SubscribeEvent(c *gin.Context) {
	identity, _ := RetrieveIdentity(c, true)
	userID := identity.UserID
	var request model.CreateEventRequest
	if err := c.ShouldBind(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: err.Error(),
		})
		return
	}

	eventTimeStart, err := time.Parse(model.EventTimeLayout, request.EventTimeStart)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: err.Error(),
		})
		return
	}
	eventTimeEnd, err := time.Parse(model.EventTimeLayout, request.EventTimeEnd)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: err.Error(),
		})
		return
	}
	newEvent := &model.Event{
		ID:             request.ID,
		EventTimeStart: &eventTimeStart,
		EventTimeEnd:   &eventTimeEnd,
		EventDetail:    request.EventDetail,
	}
	if newEvent.ID == nil {
		// Calculate event id from, event time start, event time end, event detail
		newEventId, err := model.CaculateEventID(newEvent)
		if err != nil {
			return
		}
		newEvent.ID = &newEventId
	}
	_ = ctrl.eventSvc.Store(c, newEvent)
	err = ctrl.userSvc.SubscribeEvent(c, userID, *newEvent.ID)
	switch {
	case errors.Is(err, service.ErrEventExceedsMaxSubscriptions):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, model.Response{
			Msg: "User has exceeded the maximum number of subscriptions",
		})
		return
	case errors.Is(err, service.ErrEventAlreadySubscribed):
		c.AbortWithStatusJSON(http.StatusConflict, model.Response{
			Msg: "User already subscribed to the event",
		})
		return
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	err = ctrl.asynqService.EnqueueEventNotification(c, userID, *request.ID, request.EventTimeStart)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.EventResponse{
		Data: newEvent,
	})
}

// SubscribeAllEvent godoc
// @Summary Subscribe to all events, and if remind is true, it will send all the event notification to user immediately
// @Description Subscribes a user to all events
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param remind query bool false "Send the Line notification immediately" default("true")
// @Success 200 {object} model.EventResponse "Successfully subscribed to the event"
// @Failure 400 {object} model.Response "Bad Request - Invalid input"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/events/all [post]
func (ctrl *UserController) SubscribeAllEvent(c *gin.Context) {
	identity, _ := RetrieveIdentity(c, true)
	remind, _ := strconv.ParseBool(c.Query("remind"))
	userID := identity.UserID

	events, err := ctrl.eventSvc.FindAll(c, 1, 100)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	for _, event := range events {
		err = ctrl.userSvc.SubscribeEvent(c, userID, *event.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
				Msg: err.Error(),
			})
			return
		}
	}
	if !remind {
		return
	}
	for _, event := range events {
		err := ctrl.asynqService.EnqueueEventNotification(c, userID, *event.ID, model.EventTimeLayout)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
				Msg: err.Error(),
			})
			return
		}
	}
}

// UnScribeEvent godoc
// @Summary Delete event
// @Description Deletes a specific event by its ID for a given user
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param event_id path string true "Event ID"
// @Success 200 {object} model.Response "Event successfully deleted"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/events/{event_id} [delete]
func (ctrl *UserController) UnScribeEvent(c *gin.Context) {
	identity, _ := RetrieveIdentity(c, true)
	userID := identity.UserID
	eventID := c.Param("event_id")
	err := ctrl.userSvc.UnsubscribeEvent(c, userID, eventID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	err = ctrl.asynqService.DeleteEventNotification(c, userID+eventID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
		return
	}

	c.JSON(200, model.Response{
		Msg: "Delete success",
	})
}

// GetUserSubscribeEvents godoc
// @Summary Get User Events
// @Description Retrieves a list of events associated with a user
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} model.EventListResponse "List of events associated with the user"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /users/events [get] // adjust the path and HTTP method according to your routing
func (ctrl *UserController) GetUserSubscribeEvents(c *gin.Context) {
	identity, _ := RetrieveIdentity(c, true)
	events, err := ctrl.userSvc.GetUserSubscribeEvents(c, identity.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
	}

	c.JSON(http.StatusOK, model.EventListResponse{
		Data: events,
	})
}
