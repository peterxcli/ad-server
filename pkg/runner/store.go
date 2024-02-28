package runner

import (
	"dcard-backend-2024/pkg/model"
	"sync"
	"time"
)

//TODO: use the inmem DB: https://github.com/hashicorp/go-memdb for better indexing and searching

type InMemoryStore struct {
	highestVersion int
	ads            []model.Ad
	adsByCountry   map[string][]*model.Ad
	adsByGender    map[string][]*model.Ad
	adsByPlatform  map[string][]*model.Ad
	mutex          sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		ads:           make([]model.Ad, 0),
		adsByCountry:  make(map[string][]*model.Ad),
		adsByGender:   make(map[string][]*model.Ad),
		adsByPlatform: make(map[string][]*model.Ad),
		mutex:         sync.RWMutex{},
	}
}

func (s *InMemoryStore) CreateAd(ad model.Ad) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ads = append(s.ads, ad)

	// Update indexes
	for _, country := range ad.Country {
		s.adsByCountry[country] = append(s.adsByCountry[country], &ad)
	}
	for _, gender := range ad.Gender {
		s.adsByGender[gender] = append(s.adsByGender[gender], &ad)
	}
	for _, platform := range ad.Platform {
		s.adsByPlatform[platform] = append(s.adsByPlatform[platform], &ad)
	}

	return ad.ID
}

func (s *InMemoryStore) GetAds(req GetAdRequest) ([]model.Ad, int) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Start with a larger set from one of the indexes
	candidates := map[string]*model.Ad{}
	if req.Country != "" {
		for _, ad := range s.adsByCountry[req.Country] {
			candidates[ad.ID] = ad
		}
	}
	if req.Gender != "" {
		for _, ad := range s.adsByGender[req.Gender] {
			candidates[ad.ID] = ad
		}
	}
	if req.Platform != "" {
		for _, ad := range s.adsByPlatform[req.Platform] {
			candidates[ad.ID] = ad
		}
	}

	// Now filter these candidates further based on the other criteria
	filteredAds := []model.Ad{}
	for _, ad := range candidates {
		if ad.StartAt.Before(time.Now()) && ad.EndAt.After(time.Now()) && ad.AgeStart < req.Age && req.Age < ad.AgeEnd {
			filteredAds = append(filteredAds, *ad)
		}
	}

	total := len(filteredAds)

	// Apply pagination
	start := req.offset
	if start < 0 || start >= total {
		// Return empty slice if offset is out of range
		return []model.Ad{}, 0
	}

	end := start + req.limit
	if end > total {
		end = total
	}

	paginatedAds := filteredAds[start:end]

	return paginatedAds, len(filteredAds)
}
