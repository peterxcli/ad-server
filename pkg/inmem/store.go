package inmem

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"sync"
)

var (
	// ErrNoAdsFound is returned when the ad is not found in the store, 404
	ErrNoAdsFound error = fmt.Errorf("no ads found")
	// ErrOffsetOutOfRange is returned when the offset is out of range, 404
	ErrOffsetOutOfRange error = fmt.Errorf("offset is out of range")
	// ErrInvalidVersion is returned when the version is invalid, inconsistent with the store
	ErrInvalidVersion error = fmt.Errorf("invalid version")
)

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

// DeleteAd implements model.InMemoryStore.
func (s *InMemoryStoreImpl) DeleteAd(adID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.DeleteAd(adID)
}
