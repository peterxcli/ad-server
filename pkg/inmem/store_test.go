package inmem

import (
	"dcard-backend-2024/pkg/model"
	"math/rand"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	genders = []string{"M", "F"}
	countries = []string{
        "US", "TW", "GB", "AU", "FR", "DE",
        "JP", "IN", "BR", "ZA", "CN", "RU",
        "ES", "IT", "SE", "NO", "NL", "DK",
        "MX", "AR", "PL", "BE", "FI", "NZ",
	}
	platforms = []string{"web", "ios", "android"}
)

func randRange(min, max int) int {
	if min > max {
		panic("min cannot be greater than max")
	}

	if max <= 0 {
		return rand.Intn(max - min) + min
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
	startOffset := time.Duration(randRange(-7, 0)) * 24 * time.Hour // Random start date within the last week
	endOffset := time.Duration(randRange(1, 7)) * 24 * time.Hour    // Random end date within the next week

	// Random age range ensuring AgeStart is less than AgeEnd
	ageStart := randRange(18, 65)
	ageEnd := randRange(ageStart, 65)

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
		StartAt:  time.Now().Add(startOffset),
		EndAt:    time.Now().Add(endOffset),
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
	assert.GreaterOrEqual(t, total, 0)
	assert.Len(t, ads, total)
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

	version, err := store.CreateBatchAds(ads)
	assert.Nil(t, err)
	assert.Greater(t, version, 0)
}

func TestPerformance(t *testing.T) {
	store := NewInMemoryStore()
	ads := []*model.Ad{}

	batchSize := rand.Int() % 30000 + 20000 // 20000 - 50000

	for i := 0; i < batchSize; i++ {
		ad := NewMockAd()
		ad.Version = i + 1
		ads = append(ads, ad)
	}

	start := time.Now()

	_, err := store.CreateBatchAds(ads)
	assert.Nil(t, err)

	elapsed := time.Since(start)
	averageOpsPerSecond := float64(batchSize) / elapsed.Seconds()

	if averageOpsPerSecond < 10000 {
		assert.False(t, true, "Average operations per second is too low")
	}
}

func TestReadAdsPerformanceAndAccuracy(t *testing.T) {
    store := NewInMemoryStore()

    // Populate the store with a batch of ads
    batchSize := 3000 // Adjust based on the desired test load
    for i := 0; i < batchSize; i++ {
        ad := NewMockAd()
        ad.Version = i + 1
        _, err := store.CreateAd(ad)
        assert.Nil(t, err)
    }

    testFilters := []model.GetAdRequest{
        {Age: 25, Country: "US", Gender: "", Platform: "", Offset: 0, Limit: 10},
        {Age: 30, Country: "", Gender: "F", Platform: "", Offset: 0, Limit: 10},
		{Age: 40, Country: "", Gender: "", Platform: "ios", Offset: 0, Limit: 10},
    }

    start := time.Now()

    for _, filter := range testFilters {
        ads, total, err := store.GetAds(&filter)
        assert.Nil(t, err)
        assert.NotEmpty(t, ads)
        assert.GreaterOrEqual(t, total, 0)
    }

    elapsed := time.Since(start)
    averageOpsPerSecond := float64(len(testFilters)) / elapsed.Seconds()

    assert.Greater(t, averageOpsPerSecond, 0, "The read operation is too slow")
    t.Logf("Read performance: %.2f ops/sec", averageOpsPerSecond)
}