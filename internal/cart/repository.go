package cart

import (
	"context"
	"errors"
	"time"

	gocbopentelemetry "github.com/couchbase/gocb-opentelemetry"
	"github.com/couchbase/gocb/v2"
	pb "github.com/ogozo/proto-definitions/gen/go/cart"
	"go.opentelemetry.io/otel/trace"
)

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

func createSpanOptions(ctx context.Context) *gocbopentelemetry.OpenTelemetryRequestSpan {
	span := trace.SpanFromContext(ctx)
	return gocbopentelemetry.NewOpenTelemetryRequestSpan(ctx, span)
}

func (r *Repository) GetCartByUserID(ctx context.Context, userID string) (*CartDocument, error) {
	opts := &gocb.GetOptions{
		ParentSpan: createSpanOptions(ctx),
	}

	getResult, err := r.collection.Get(userID, opts)
	if err != nil {
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			newCart := &CartDocument{
				UserID:        userID,
				Items:         []*pb.CartItem{},
				LastUpdatedAt: time.Now().UTC(),
			}
			if err := r.UpdateCart(ctx, newCart); err != nil {
				return nil, err
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

func (r *Repository) UpdateCart(ctx context.Context, cart *CartDocument) error {
	cart.LastUpdatedAt = time.Now().UTC()
	opts := &gocb.UpsertOptions{
		ParentSpan: createSpanOptions(ctx),
	}
	_, err := r.collection.Upsert(cart.UserID, cart, opts)
	return err
}

func (r *Repository) ClearCart(ctx context.Context, userID string) error {
	opts := &gocb.RemoveOptions{
		ParentSpan: createSpanOptions(ctx),
	}
	_, err := r.collection.Remove(userID, opts)
	if err != nil && !errors.Is(err, gocb.ErrDocumentNotFound) {
		return err
	}
	return nil
}
