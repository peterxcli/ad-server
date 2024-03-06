package inmem

import (
	"dcard-backend-2024/pkg/model"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	genders   = []string{"M", "F"}
	countries = []string{
		"US", "TW", "GB", "AU", "FR", "DE",
		"JP", "IN", "BR", "ZA", "CN", "RU",
		"ES", "IT", "SE", "NO", "NL", "DK",
		"MX", "AR", "PL", "BE", "FI", "NZ",
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
	startOffset := time.Duration(randRange(-7, 0)) * 24 * time.Hour
	endOffset := time.Duration(randRange(1, 7)) * 24 * time.Hour

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
		Title:    faker.Sentence(),
		Content:  faker.Paragraph(),
		StartAt:  model.CustomTime(time.Now().Add(startOffset)),
		EndAt:    model.CustomTime(time.Now().Add(endOffset)),
		AgeStart: ageStart,
		AgeEnd:   ageEnd,
		Gender:   genderSelection,
		Country:  countrySelection,
		Platform: platformSelection,
	}
}

func TestCreateAd(t *testing.T) {
	store := NewInMemoryStore()
	ad := NewMockAd()
	ad.Version = 1

	id, err := store.CreateAd(ad)
	assert.Equal(t, ad.ID.String(), id)
	assert.Nil(t, err)
	assert.NotEmpty(t, id)
}

func TestGetAds(t *testing.T) {
	store := NewInMemoryStore()
	ad := NewMockAd()
	ad.Version = 1
	_, err := store.CreateAd(ad)
	assert.Nil(t, err)

	request := &model.GetAdRequest{
		Age:      randRange(ad.AgeStart, ad.AgeEnd),
		Country:  ad.Country[0],
		Gender:   ad.Gender[0],
		Platform: ad.Platform[0],
		Offset:   0,
		Limit:    10,
	}

	ads, total, err := store.GetAds(request)
	assert.Nil(t, err)
	assert.Equal(t, total, 1)
	assert.Equal(t, ads[0].ID, ad.ID)
	assert.Len(t, ads, total)
}

func TestGetNoAds(t *testing.T) {
	store := NewInMemoryStore()
	ad := NewMockAd()
	ad.Version = 1
	_, err := store.CreateAd(ad)
	assert.Nil(t, err)

	request := &model.GetAdRequest{
		Age:      randRange(ad.AgeStart, ad.AgeEnd),
		Country:  ad.Country[0],
		Gender:   ad.Gender[0],
		Platform: "1",
		Offset:   0,
		Limit:    10,
	}

	_, _, err = store.GetAds(request)
	assert.NotNil(t, err)
}

func TestCreateBatchAds(t *testing.T) {
	store := NewInMemoryStore()
	ads := []*model.Ad{}

	batchSize := rand.Int() % 1000

	for i := 0; i < batchSize; i++ {
		ad := NewMockAd()
		ad.Version = i + 1
		ads = append(ads, ad)
	}

	err := store.CreateBatchAds(ads)
	assert.Nil(t, err)
}

func TestCreatePerformance(t *testing.T) {
	store := NewInMemoryStore()
	ads := []*model.Ad{}

	batchSize := rand.Int()%30000 + 20000 // 20000 - 50000

	for i := 0; i < batchSize; i++ {
		ad := NewMockAd()
		ad.Version = i + 1
		ads = append(ads, ad)
	}

	start := time.Now()

	err := store.CreateBatchAds(ads)
	assert.Nil(t, err)

	elapsed := time.Since(start)
	averageOpsPerSecond := float64(batchSize) / elapsed.Seconds()
	t.Logf("Create performance: %.2f ops/sec", averageOpsPerSecond)
	if averageOpsPerSecond < 10000 {
		assert.False(t, true, "Average operations per second is too low")
	}
}

func generateRandomGetAdRequest() model.GetAdRequest {
	age := randRange(18, 65)
	country := countries[rand.Intn(len(countries))]
	gender := genders[rand.Intn(len(genders))]
	platform := platforms[rand.Intn(len(platforms))]

	return model.GetAdRequest{
		Age:      age,
		Country:  country,
		Gender:   gender,
		Platform: platform,
		Offset:   0,
		Limit:    randRange(10, 10),
	}
}
func TestReadAdsPerformanceAndAccuracy(t *testing.T) {
	store := NewInMemoryStore()

	// Populate the store with a batch of ads
	batchSize := 3000
	for i := 0; i < batchSize; i++ {
		ad := NewMockAd()
		ad.Version = i + 1
		_, err := store.CreateAd(ad)
		assert.Nil(t, err)
	}

	queryCount := randRange(10000, 20000)
	testFilters := []model.GetAdRequest{}
	for i := 0; i < queryCount; i++ {
		testFilters = append(testFilters, generateRandomGetAdRequest())
	}
	testFilters = append(testFilters,
		// Only Country has a value
		model.GetAdRequest{
			Age:      18,
			Country:  "US",
			Gender:   "",
			Platform: "",
			Offset:   0,
			Limit:    10,
		},
		// Only Gender has a value
		model.GetAdRequest{
			Age:      18,
			Country:  "",
			Gender:   "F",
			Platform: "",
			Offset:   0,
			Limit:    10,
		},
		// Only Platform has a value
		model.GetAdRequest{
			Age:      18,
			Country:  "",
			Gender:   "",
			Platform: "ios",
			Offset:   0,
			Limit:    10,
		},
		// no value
		model.GetAdRequest{
			Age:      18,
			Country:  "",
			Gender:   "",
			Platform: "",
			Offset:   0,
			Limit:    10,
		},
	)

	start := time.Now()
	wg := sync.WaitGroup{}

	for _, filter := range testFilters {
		wg.Add(1)
		go func(filter model.GetAdRequest) {
			defer wg.Done()
			store.GetAds(&filter)
		}(filter)
	}

	wg.Wait()

	elapsed := time.Since(start)
	averageOpsPerSecond := float64(len(testFilters)) / elapsed.Seconds()

	assert.Greater(t, int(averageOpsPerSecond), 10000, "The read operation is too slow")
	t.Logf("Read performance: %.2f ops/sec", averageOpsPerSecond)
}

func BenchmarkReadAds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestReadAdsPerformanceAndAccuracy(&testing.T{})
	}
}
