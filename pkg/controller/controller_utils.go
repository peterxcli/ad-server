package controller

import (
	"bikefest/pkg/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RetrievePagination(c *gin.Context) (page, limit uint64) {
	pageStr := c.Query("page")

	limitStr := c.Query("limit")

	// Convert string to uint64
	page, err := strconv.ParseUint(pageStr, 10, 64)
	if err != nil {
		page = 1
	}

	limit, err = strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		limit = 10
	}

	return
}

// RetrieveIdentity retrieves the identity of the user from the context.
// raise: Raise a http error when the identity doesn't exist.
func RetrieveIdentity(c *gin.Context, raise bool) (identity *model.Identity, exist bool) {
	id, exist := c.Get("identity")
	if !exist {
		if raise {
			c.AbortWithStatusJSON(401, model.Response{
				Msg: "Login Required",
			})
		}
		return nil, false
	}
	identity = id.(*model.Identity)
	return
}
