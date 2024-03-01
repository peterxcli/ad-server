package controller

import (
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
