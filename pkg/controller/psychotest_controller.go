package controller

import (
	"bikefest/pkg/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PsychoTestController struct {
	db *gorm.DB
}

func NewPsychoTestController(db *gorm.DB) *PsychoTestController {
	return &PsychoTestController{db: db}
}

// Function: CreateType
// The function is responsible for creating
// new psychological type
func (controller *PsychoTestController) CreateType(context *gin.Context) {
	newType := context.Query("type")

	if newType == "" {
		context.JSON(http.StatusBadRequest, gin.H{
			"status":  "Failed",
			"message": "Missing or invalid parameters",
		})
		return
	}

	record := model.PsychoTest{
		Type:  newType,
		Count: 0,
	}

	if result := controller.db.Create(&record); result.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{
			"status":  "Failed",
			"message": result.Error,
		})
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"status":  "Success",
		"message": "Create type successfully",
	})
}

// Function: TypeAddCount
// The function is responsible for adding
// the count of selected psychological type
func (controller *PsychoTestController) TypeAddCount(context *gin.Context) {
	testType := context.PostForm("type")
	count, _ := strconv.Atoi(context.PostForm("count"))

	if testType == "" || count == 0 {
		context.JSON(http.StatusBadRequest, gin.H{
			"status":  "Failed",
			"message": "Missing or invalid parameters",
		})
		return
	}

	var psychoType *model.PsychoTest

	controller.db.Where("type = ?", testType).First(&psychoType)

	if psychoType == nil {
		context.JSON(http.StatusNotFound, gin.H{
			"status":  "Failed",
			"message": "Psychological type doesn't exist",
		})
		return
	}

	psychoType.Count += count
	controller.db.Save(&psychoType)
	context.JSON(http.StatusOK, gin.H{
		"status":  "Success",
		"message": "Successfully add the count of the type",
	})
}

// Function: CountTypePercentage
// The function is responsible for retreval of
// of the percentage of each type
func (controller *PsychoTestController) CountTypePercentage(context *gin.Context) {
	var queryTypes []*model.PsychoTest

	controller.db.Find(&queryTypes)

	if len(queryTypes) == 0 {
		context.JSON(http.StatusBadRequest, gin.H{
			"status":  "Failed",
			"message": "No existing psychological test",
		})
		return
	}

	psychoTypes := make(map[string]float64, len(queryTypes))
	sum := 0

	for _, t := range queryTypes {
		sum += t.Count
	}

	if sum == 0 {
		context.JSON(http.StatusNotFound, gin.H{
			"status":  "Failed",
			"message": "No test data",
		})
	}

	for _, t := range queryTypes {
		psychoTypes[t.Type] = float64(t.Count) / float64(sum) * 100
	}

	context.JSON(http.StatusOK, gin.H{
		"status": "Success",
		"data":   psychoTypes,
	})
}
