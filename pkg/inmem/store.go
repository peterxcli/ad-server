package inmem

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"sync"

	"github.com/wangjia184/sortedset"
)

var (
	// ErrNoAdsFound is returned when the ad is not found in the store, 404
	ErrNoAdsFound error = fmt.Errorf("no ads found")
	// ErrOffsetOutOfRange is returned when the offset is out of range, 404
	ErrOffsetOutOfRange error = fmt.Errorf("offset is out of range")
	// ErrInvalidVersion is returned when the version is invalid, inconsistent with the store
	ErrInvalidVersion error = fmt.Errorf("invalid version")
)

type QueryIndex struct {
	// Ages maps a string representation of an age range to a CountryIndex
	Ages map[uint8]*CountryIndex
}

type CountryIndex struct {
	// Countries maps country codes to PlatformIndex
	Countries map[string]*PlatformIndex
}

type PlatformIndex struct {
	// Platforms maps platform names to GenderIndex
	Platforms map[string]*GenderIndex
}

type GenderIndex struct {
	// Genders maps gender identifiers to sets of Ad IDs
	Genders map[string]*sortedset.SortedSet
}

type AdIndex interface {
	// AddAd adds an ad to the index
	AddAd(ad *model.Ad) error
	// RemoveAd removes an ad from the index
	RemoveAd(ad *model.Ad) error
	// GetAdIDs returns the ad IDs that match the given query
	GetAdIDs(req *model.GetAdRequest) ([]*model.Ad, error)
}

type AdIndexImpl struct {
	// index is the root index
	index *QueryIndex
}

// AddAd implements AdIndex.
func (a *AdIndexImpl) AddAd(ad *model.Ad) error {
	targetCountries := append(ad.Country, "")
	targetPlatforms := append(ad.Platform, "")
	targetGenders := append(ad.Gender, "")
	targetAges := []uint8{0}
	for age := ad.AgeStart; age <= ad.AgeEnd; age++ {
		targetAges = append(targetAges, age)
	}
	for _, country := range targetCountries {
		for _, platform := range targetPlatforms {
			for _, gender := range targetGenders {
				for _, age := range targetAges {
					ageIndex, ok := a.index.Ages[age]
					if !ok {
						ageIndex = &CountryIndex{Countries: make(map[string]*PlatformIndex)}
						a.index.Ages[age] = ageIndex
					}

					platformIndex, ok := ageIndex.Countries[country]
					if !ok {
						platformIndex = &PlatformIndex{Platforms: make(map[string]*GenderIndex)}
						ageIndex.Countries[country] = platformIndex
					}

					genderIndex, ok := platformIndex.Platforms[platform]
					if !ok {
						genderIndex = &GenderIndex{Genders: make(map[string]*sortedset.SortedSet)}
						platformIndex.Platforms[platform] = genderIndex
					}

					adSet, ok := genderIndex.Genders[gender]
					if !ok {
						adSet = sortedset.New()
						genderIndex.Genders[gender] = adSet
					}
					adSet.AddOrUpdate(ad.ID.String(), sortedset.SCORE(ad.StartAt.T().Unix()), ad)
				}
			}
		}
	}
	return nil
}

// GetAdIDs implements AdIndex.
func (a *AdIndexImpl) GetAdIDs(req *model.GetAdRequest) ([]*model.Ad, error) {
	ageIndex, ok := a.index.Ages[req.Age]
	if !ok {
		return nil, ErrNoAdsFound
	}

	platformIndex, ok := ageIndex.Countries[req.Country]
	if !ok {
		return nil, ErrNoAdsFound
	}

	genderIndex, ok := platformIndex.Platforms[req.Platform]
	if !ok {
		return nil, ErrNoAdsFound
	}

	adSet, ok := genderIndex.Genders[req.Gender]
	if !ok {
		return nil, ErrNoAdsFound
	}

	// get the ad IDs from the sorted set
	result := adSet.GetByRankRange(req.Offset, req.Offset+req.Limit, false)

	ads := make([]*model.Ad, 0, len(result))
	for _, ad := range result {
		ads = append(ads, ad.Value.(*model.Ad))
	}
	return ads, nil
}

// RemoveAd implements AdIndex.
func (a *AdIndexImpl) RemoveAd(ad *model.Ad) error {
	panic("unimplemented")
}

func NewAdIndex() AdIndex {
	return &AdIndexImpl{
		index: &QueryIndex{
			Ages: make(map[uint8]*CountryIndex),
		},
	}
}

// InMemoryStoreImpl is an in-memory ad store implementation
type InMemoryStoreImpl struct {
	// ads maps ad IDs to ads
	ads     map[string]*model.Ad
	adIndex AdIndex
	mutex   sync.RWMutex
}

func NewInMemoryStore() model.InMemoryStore {
	return &InMemoryStoreImpl{
		ads:     make(map[string]*model.Ad),
		adIndex: NewAdIndex(),
		mutex:   sync.RWMutex{},
	}
}

// CreateBatchAds creates a batch of ads in the store
// (only used in the snapshot restore)
func (s *InMemoryStoreImpl) CreateBatchAds(ads []*model.Ad) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, ad := range ads {
		s.ads[ad.ID.String()] = ad
		_ = s.adIndex.AddAd(ad)
	}
	return nil
}

func (s *InMemoryStoreImpl) CreateAd(ad *model.Ad) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ads[ad.ID.String()] = ad
	_ = s.adIndex.AddAd(ad)
	return ad.ID.String(), nil
}

func (s *InMemoryStoreImpl) GetAds(req *model.GetAdRequest) (ads []*model.Ad, count int, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ads, err = s.adIndex.GetAdIDs(req)
	if err != nil {
		return nil, 0, err
	}
	return ads, len(ads), nil
}
