package inmem

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	// ErrNoAdsFound is returned when the ad is not found in the store, 404
	ErrNoAdsFound error = fmt.Errorf("no ads found")
	// ErrOffsetOutOfRange is returned when the offset is out of range, 404
	ErrOffsetOutOfRange error = fmt.Errorf("offset is out of range")
	// ErrInvalidVersion is returned when the version is invalid, inconsistent with the store
	ErrInvalidVersion error = fmt.Errorf("invalid version")
)

type InMemoryStoreImpl struct {
	// use the Version as redis stream's message sequence number, and also store it as ad's version
	// then if the rebooted service's version is lower than the Version, it will fetch the latest ads from the db
	// and use the db's version as the Version, then start subscribing the redis stream from the Version offset
	Version       int
	ads           map[string]*model.Ad
	adsByCountry  map[string]map[string]*model.Ad
	adsByGender   map[string]map[string]*model.Ad
	adsByPlatform map[string]map[string]*model.Ad
	mutex         sync.RWMutex
}

func NewInMemoryStore() model.InMemoryStore {
	return &InMemoryStoreImpl{
		ads:           make(map[string]*model.Ad),
		adsByCountry:  make(map[string]map[string]*model.Ad),
		adsByGender:   make(map[string]map[string]*model.Ad),
		adsByPlatform: make(map[string]map[string]*model.Ad),
		mutex:         sync.RWMutex{},
	}
}

// CreateBatchAds creates a batch of ads in the store
// this function does not check the version continuity.
// because if we want to support update operation restore from the snapshot,
// the version must not be continuous
// (only used in the snapshot restore)
func (s *InMemoryStoreImpl) CreateBatchAds(ads []*model.Ad) (version int, err error) {
	// sort the ads by version
	sort.Slice(ads, func(i, j int) bool {
		return ads[i].Version < ads[j].Version
	})

	for _, ad := range ads {
		// Update the version
		s.Version = max(s.Version, ad.Version)

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
	}
	return s.Version, nil
}

func (s *InMemoryStoreImpl) CreateAd(ad *model.Ad) (string, error) {
	if ad.Version != s.Version+1 {
		return "", ErrInvalidVersion
	}

	// Update the version
	s.Version = ad.Version

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

func (s *InMemoryStoreImpl) GetAds(req *model.GetAdRequest) (ads []*model.Ad, count int, err error) {
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
	// TODO: use a B+ tree to index the ads by StartAt and EndAt and AgeStart and AgeEnd
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
