package controller

import (
	"dcard-backend-2024/pkg/inmem"
	"dcard-backend-2024/pkg/model"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdController struct {
	adService model.AdService
}

func NewAdController(adService model.AdService) *AdController {
	return &AdController{
		adService: adService,
	}
}

// GetAd godoc
// @Summary Get an ad by ID
// @Description Retrieves an ad by ID
// @Tags Ad
// @Accept json
// @Produce json
// @Param offset query int false "Offset for pagination"
// @Param limit query int false "Limit for pagination"
// @Param age query int false "Age"
// @Param gender query string false "Gender"
// @Param country query string false "Country"
// @Param platform query string false "Platform"
// @Success 200 {object} model.GetAdsPageResponse
// @Failure 404 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/v1/ad [get]
func (ac *AdController) GetAd(c *gin.Context) {
	var req model.GetAdRequest
	if err := c.BindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Msg: err.Error()})
		return
	}

	ads, total, err := ac.adService.GetAds(c, &req)
	switch {
	case errors.Is(err, inmem.ErrNoAdsFound):
		c.JSON(http.StatusNotFound, model.Response{Msg: err.Error()})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, model.Response{Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.GetAdsPageResponse{Ads: ads, Total: total})
}

// CreateAd godoc
// @Summary Create an ad
// @Description Create an ad
// @Tags Ad
// @Accept json
// @Produce json
// @Param ad body model.CreateAdRequest true "Ad object"
// @Success 201 {object} model.CreateAdResponse
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/v1/ad [post]
func (ac *AdController) CreateAd(c *gin.Context) {
	var ad model.CreateAdRequest
	if err := c.BindJSON(&ad); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Msg: err.Error()})
		return
	}

	adID, err := ac.adService.CreateAd(c,
		&model.Ad{
			Title:    ad.Title,
			Content:  ad.Content,
			StartAt:  ad.StartAt,
			EndAt:    ad.EndAt,
			AgeStart: ad.AgeStart,
			AgeEnd:   ad.AgeEnd,
			Gender:   ad.Gender,
			Country:  ad.Country,
			Platform: ad.Platform,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Msg: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model.Response{Msg: "Ad created", Data: adID})
}
