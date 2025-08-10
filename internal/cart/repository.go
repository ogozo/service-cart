package cart

import (
	"errors"
	"time"

	"github.com/couchbase/gocb/v2"
	pb "github.com/ogozo/proto-definitions/gen/go/cart"
)

// Dökümanımızın Couchbase'deki yapısı
type CartDocument struct {
	UserID        string         `json:"userId"`
	Items         []*pb.CartItem `json:"items"`
	LastUpdatedAt time.Time      `json:"lastUpdatedAt"`
}

type Repository struct {
	collection *gocb.Collection
}

func NewRepository(collection *gocb.Collection) *Repository {
	return &Repository{collection: collection}
}

func (r *Repository) GetCartByUserID(userID string) (*CartDocument, error) {
	getResult, err := r.collection.Get(userID, nil)

	if err != nil {
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			newCart := &CartDocument{
				UserID:        userID,
				Items:         []*pb.CartItem{},
				LastUpdatedAt: time.Now().UTC(),
			}
			_, upsertErr := r.collection.Upsert(userID, newCart, nil)
			if upsertErr != nil {
				return nil, upsertErr
			}
			return newCart, nil
		}
		return nil, err
	}

	var cart CartDocument
	if err := getResult.Content(&cart); err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *Repository) UpdateCart(userID string, cart *CartDocument) error {
	cart.LastUpdatedAt = time.Now().UTC()
	_, err := r.collection.Upsert(userID, cart, nil)
	return err
}
