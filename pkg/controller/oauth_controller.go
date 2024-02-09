package controller

import (
	"bikefest/pkg/bootstrap"
	"bikefest/pkg/line_utils"
	"bikefest/pkg/model"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	social "github.com/kkdai/line-login-sdk-go"
)

func NewOAuthController(lineSocialClient *social.Client, env *bootstrap.Env, userSvc model.UserService) *OAuthController {
	return &OAuthController{
		lineSocialClient: lineSocialClient,
		userSvc:          userSvc,
		env:              env,
	}
}

type OAuthController struct {
	lineSocialClient *social.Client
	userSvc          model.UserService
	env              *bootstrap.Env
}

// LineLogin initiates a login process using LINE's OAuth service.
// @Summary Initiate LINE OAuth login
// @Description Redirects the user to LINE's OAuth service for authentication.
// @Tags OAuth
// @Accept  json
// @Produce  json
// @Param redirect_path query string false "Redirect path after login"
// @Success 301 {string} string "Redirect to the target URL"
// @Failure 400 {string} string "Bad Request"
// @Router /line-login/auth [get]
func (ctrl *OAuthController) LineLogin(c *gin.Context) {
	redirectedPath := c.Query("redirect_path")
	originalUrl := c.Request.Referer()
	if len(redirectedPath) != 0 {
		originalUrl += redirectedPath[1:]
	}

	log.Println("originalUrl:", originalUrl)
	serverURL := ctrl.env.Line.ServerUrl
	scope := "profile openid" //profile | openid | email
	state := originalUrl + "$" + social.GenerateNonce()
	if len(state) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: "Login with the wrong way, please try again in official website",
		})
		return
	}
	nonce := social.GenerateNonce()
	redirectURL := fmt.Sprintf("%s/line-login/callback", serverURL)
	targetURL := ctrl.lineSocialClient.GetWebLoinURL(redirectURL, state, scope, social.AuthRequestOptions{Nonce: nonce, Prompt: "consent", BotPrompt: "aggressive"})
	c.SetCookie("state", state, 3600, "/", "", false, true)
	c.Redirect(http.StatusFound, targetURL)
}

// LineLoginCallback handles the callback from LINE's OAuth service.
// @Summary Handle LINE OAuth callback
// @Description Handles the callback from LINE's OAuth service and redirects the user to the frontend with the tokens in the query and cookies.
// @Tags OAuth
// @Accept  json
// @Produce  json
// @Param code query string true "Authorization code"
// @Param state query string true "State"
// @Success 301 {string} string "Redirect to the frontend"
// @Failure 400 {string} string "Bad Request"
// @Router /line-login/callback [get]
func (ctrl *OAuthController) LineLoginCallback(c *gin.Context) {
	serverURL := ctrl.env.Line.ServerUrl
	code := c.Query("code")
	state := c.Query("state")
	stateInCookie, err := c.Cookie("state")
	if err != nil || stateInCookie != state {
		c.AbortWithStatusJSON(http.StatusBadRequest, model.Response{
			Msg: "State cookie is invalid",
		})
		return
	}
	log.Println("code:", code, " stateInCookie:", stateInCookie)
	frontendURL := strings.Split(stateInCookie, "$")[0]
	token, err := ctrl.lineSocialClient.GetAccessToken(fmt.Sprintf("%s/line-login/callback", serverURL), code).Do()
	if err != nil {
		log.Println("RequestLoginToken err:", err)
		return
	}
	log.Println("access_token:", token.AccessToken, " refresh_token:", token.RefreshToken)

	// check friendship with official account
	friendFlag, err := line_utils.GetFriendshipStatus(token.AccessToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
			Msg: err.Error(),
		})
	}
	if friendFlag != true {
		c.AbortWithStatusJSON(http.StatusForbidden, model.Response{
			Msg: "You are not a friend of the official account",
		})
		return
	}
	var payload *social.Payload
	payload, err = token.DecodePayload(ctrl.env.Line.ChannelID)
	if err != nil {
		log.Println("DecodeIDToken err:", err)
		return
	}
	log.Printf("payload: %#v", payload)

	user := &model.User{
		ID:   payload.Sub,
		Name: payload.Name,
	}

	err = ctrl.userSvc.CreateFakeUser(c, user)

	if err != nil {
		log.Printf("user with id %s already exists", user.ID)
	}

	accessToken, err := ctrl.userSvc.CreateAccessToken(c, user, ctrl.env.JWT.AccessTokenSecret, ctrl.env.JWT.AccessTokenExpiry)
	if err != nil {
		log.Printf("failed to create access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Failed",
			"message": "failed to create access token",
		})
		return
	}

	refreshToken, err := ctrl.userSvc.CreateRefreshToken(c, user, ctrl.env.JWT.RefreshTokenSecret, ctrl.env.JWT.RefreshTokenExpiry)
	if err != nil {
		log.Printf("failed to create refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Failed",
			"message": "failed to create refresh token",
		})
		return
	}

	// set to cookie
	c.SetCookie("access_token", fmt.Sprintf("Bearer %s", accessToken), 3600, "/", "", false, true)
	c.SetCookie("refresh_token", fmt.Sprintf("Bearer %s", refreshToken), 3600, "/", "", false, true)
	// redirect to frontend
	log.Println("redirect to frontend:", frontendURL)
	c.Redirect(http.StatusFound, fmt.Sprintf("%s?access_token=%s&refresh_token=%s", frontendURL, accessToken, refreshToken))
}
