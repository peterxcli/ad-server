package main

import (
	"dcard-backend-2024/pkg/bootstrap"
	"dcard-backend-2024/pkg/model"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

var (
	genders   = []string{"M", "F"}
	countries = []string{
		"US", "TW", "GB", "AU", "FR", "DE",
	}
	platforms = []string{"web", "ios", "android"}
)

func randRange(min, max int) int {
	if min == max {
		return min
	}
	if min > max {
		panic("min cannot be greater than max")
	}

	if max <= 0 {
		return rand.Intn(max-min) + min
	}
	return rand.Intn(max-min) + min
}

func shuffle(arr []string, inplace bool) []string {
	if inplace {
		for i := range arr {
			j := rand.Intn(i + 1)
			arr[i], arr[j] = arr[j], arr[i]
		}
		return arr
	}

	result := make([]string, len(arr))
	copy(result, arr)
	for i := range result {
		j := rand.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func NewMockAd() *model.Ad {
	startOffset := time.Duration(-1) * 24 * 30 * 365 * time.Hour
	endOffset := time.Duration(1) * 24 * 30 * 365 * time.Hour

	// Random age range ensuring AgeStart is less than AgeEnd
	ageStart := randRange(18, 63)
	ageEnd := randRange(ageStart+1, 65)

	// Random selection of genders

	gender_shuffled := shuffle(genders, false)
	genderSelection := gender_shuffled[:randRange(1, 2)]

	countries_shuffled := shuffle(countries, false)
	countrySelection := countries_shuffled[:randRange(1, len(countries_shuffled))]

	platforms_shuffled := shuffle(platforms, false)
	platformSelection := platforms_shuffled[:randRange(1, len(platforms))]

	return &model.Ad{
		ID:       uuid.New(),
		Title:    "",
		Content:  "",
		StartAt:  model.CustomTime(time.Now().Add(startOffset)),
		EndAt:    model.CustomTime(time.Now().Add(endOffset)),
		AgeStart: uint8(ageStart),
		AgeEnd:   uint8(ageEnd),
		Gender:   genderSelection,
		Country:  countrySelection,
		Platform: platformSelection,
	}
}

func main() {
	// Init config
	app := bootstrap.App()

	// Create mock data
	batchNum := 1000 // Active ads < 1000
	data := make([]*model.Ad, batchNum)
	for i := 0; i < batchNum; i++ {
		data[i] = NewMockAd()
		data[i].Version = i + 1
	}

	// jump out a prompt to confirm
	fmt.Println("Are you sure to inject mock data? (y/n)")
	var input string
	fmt.Scanln(&input)
	if input != "y" {
		return
	}
	app.Conn.Exec("TRUNCATE TABLE ads")
	app.Conn.Model(&model.Ad{}).Create(&data)
}
