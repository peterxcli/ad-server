package inmem

import (
	"dcard-backend-2024/pkg/model"
	"fmt"
	"log"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/wangjia184/sortedset"
)

type IndexNode interface {
	AddAd(ad *model.Ad)
	GetAd(req *model.GetAdRequest) ([]*model.Ad, error)
	DeleteAd(ad *model.Ad)
}

type FieldStringer struct {
	Value interface{}
}

func (f FieldStringer) String() string {
	return fmt.Sprintf("%v", f.Value)
}

func (g *IndexInternalNode) AddAd(ad *model.Ad) {
	values, err := ad.GetValueByKey(g.Key)
	if err != nil {
		log.Printf("AddAd: Error getting value by key \"%s\": %s\n", g.Key, err)
		return
	}

	var wg sync.WaitGroup

	for _, v := range values {
		wg.Add(1)
		go func(v interface{}) {
			defer wg.Done()

			field := FieldStringer{Value: v}

			child, exists := g.Children.Get(field)
			if !exists {
				nextKey := model.Ad{}.GetNextIndexKey(g.Key)
				if nextKey == "" {
					child = NewIndexLeafNode()
				} else {
					child = NewIndexInternalNode(nextKey)
				}
				g.Children.Set(field, child)
			}

			child.AddAd(ad)
		}(v)
	}

	wg.Wait()
}

// GetAd implements IndexNode.
func (g *IndexInternalNode) GetAd(req *model.GetAdRequest) ([]*model.Ad, error) {
	values, err := req.GetValueByKey(g.Key)
	if err != nil {
		return nil, fmt.Errorf("GetAd: Error getting value by key \"%s\": %s", g.Key, err)
	}

	Field := FieldStringer{Value: values}
	child, exists := g.Children.Get(Field)
	if !exists {
		return nil, nil
	}

	ads, err := child.GetAd(req)
	return ads, nil
}

// DeleteAd implements IndexNode.
func (g *IndexInternalNode) DeleteAd(ad *model.Ad) {
	values, err := ad.GetValueByKey(g.Key)
	if err != nil {
		log.Printf("Error getting value by key \"%s\": %s\n", g.Key, err)
		return
	}

	for _, v := range values {
		field := FieldStringer{Value: v}
		child, exists := g.Children.Get(field)
		if !exists {
			continue
		}

		child.DeleteAd(ad)
	}
}

type IndexInternalNode struct {
	Key      string                                       // The key this node indexes on, e.g., "country", "age"
	Children cmap.ConcurrentMap[FieldStringer, IndexNode] // The children of this node
}

func NewIndexInternalNode(key string) IndexNode {
	return &IndexInternalNode{
		Key:      key,
		Children: cmap.NewStringer[FieldStringer, IndexNode](),
	}
}

type IndexLeafNode struct {
	mu  sync.RWMutex
	Ads *sortedset.SortedSet // map[string]*model.Ad
}

func (g *IndexLeafNode) AddAd(ad *model.Ad) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Ads.AddOrUpdate(ad.ID.String(), sortedset.SCORE(ad.CreatedAt.T().Unix()), ad)
}

// GetAd implements IndexNode.
func (g *IndexLeafNode) GetAd(req *model.GetAdRequest) ([]*model.Ad, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ad := g.Ads.GetByRankRange(req.Offset, req.Offset+req.Limit, false)
	ret := make([]*model.Ad, len(ad))
	for i, a := range ad {
		ret[i] = a.Value.(*model.Ad)
	}
	return ret, nil
}

// DeleteAd implements IndexNode.
func (g *IndexLeafNode) DeleteAd(ad *model.Ad) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Ads.Remove(ad.ID.String())
}

func NewIndexLeafNode() IndexNode {
	return &IndexLeafNode{
		Ads: sortedset.New(),
	}
}
