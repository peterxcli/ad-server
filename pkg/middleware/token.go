package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"

	tokensvc "bikefest/internal/token"
	"bikefest/pkg/model"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(accessSecret string, cache *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// get token from cookie
			authHeader, _ = c.Cookie("access_token")
		}
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Msg: "Invalid token format (length different from 2)",
			})
			return
		}
		authToken := bearerToken[1]
		claims, err := tokensvc.ExtractCustomClaimsFromToken(&authToken, accessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Msg: err.Error(),
			})
			return
		}
		exists, err := cache.Exists(c, claims.ID).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, model.Response{
				Msg: fmt.Errorf("failed to check token existence: cache server error").Error(),
			})
			return
		}
		if exists == 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Msg: fmt.Errorf("token has been revoked").Error(),
			})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{
				Msg: err.Error(),
			})
			return
		}
		identity := &claims.Identity
		c.Set("identity", identity)
		c.Next()
	}
}
