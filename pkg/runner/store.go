package runner

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrNoAdsFound is returned when the ad is not found in the store, 404
	ErrNoAdsFound error = fmt.Errorf("no ads found")
	// ErrOffsetOutOfRange is returned when the offset is out of range, 404
	ErrOffsetOutOfRange error = fmt.Errorf("offset is out of range")
)

//TODO: use the inmem DB: https://github.com/hashicorp/go-memdb for better indexing and searching

type InMemoryStore struct {
	highestVersion int
	ads            map[string]*model.Ad
	adsByCountry   map[string]map[string]*model.Ad
	adsByGender    map[string]map[string]*model.Ad
	adsByPlatform  map[string]map[string]*model.Ad
	mutex          sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		ads:           make(map[string]*model.Ad),
		adsByCountry:  make(map[string]map[string]*model.Ad),
		adsByGender:   make(map[string]map[string]*model.Ad),
		adsByPlatform: make(map[string]map[string]*model.Ad),
		mutex:         sync.RWMutex{},
	}
}

func (s *InMemoryStore) CreateAd(ad *model.Ad) (string, error) {
	// s.mutex.Lock()
	// defer s.mutex.Unlock()

	s.ads[ad.ID] = ad

	// Update indexes
	for _, country := range ad.Country {
		if s.adsByCountry[country] == nil {
			s.adsByCountry[country] = make(map[string]*model.Ad)
		}
		s.adsByCountry[country][ad.ID] = ad
	}
	for _, gender := range ad.Gender {
		if s.adsByGender[gender] == nil {
			s.adsByGender[gender] = make(map[string]*model.Ad)
		}
		s.adsByGender[gender][ad.ID] = ad
	}
	for _, platform := range ad.Platform {
		if s.adsByPlatform[platform] == nil {
			s.adsByPlatform[platform] = make(map[string]*model.Ad)
		}
		s.adsByPlatform[platform][ad.ID] = ad
	}

	return ad.ID, nil
}

func (s *InMemoryStore) GetAds(req *GetAdRequest) (ads []*model.Ad, count int, err error) {
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()

	// Start with a larger set from one of the indexes
	candidates := map[string]*model.Ad{}
	if req.Country != "" {
		for id, ad := range s.adsByCountry[req.Country] {
			candidates[id] = ad
		}
	}
	if req.Gender != "" {
		for id, ad := range s.adsByGender[req.Gender] {
			candidates[id] = ad
		}
	}
	if req.Platform != "" {
		for id, ad := range s.adsByPlatform[req.Platform] {
			candidates[id] = ad
		}
	}

	// Now filter these candidates further based on the other criteria
	filteredAds := []*model.Ad{}
	for _, ad := range candidates {
		now := time.Now()
		if ad.StartAt.Before(now) && ad.EndAt.After(now) && ad.AgeStart <= req.Age && req.Age <= ad.AgeEnd {
			filteredAds = append(filteredAds, ad)
		}
	}

	total := len(filteredAds)

	if total == 0 {
		return nil, 0, ErrNoAdsFound
	}

	// Apply pagination
	start := req.Offset
	if start < 0 || start >= total {
		// Return empty slice if offset is out of range
		return nil, 0, ErrOffsetOutOfRange
	}

	end := start + req.Limit
	if end > total {
		end = total
	}

	paginatedAds := filteredAds[start:end]

	return paginatedAds, total, nil
}
