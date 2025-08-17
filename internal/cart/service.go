package cart

import (
	"context"

	pb "github.com/ogozo/proto-definitions/gen/go/cart"
	"github.com/ogozo/service-cart/internal/broker"
	"github.com/ogozo/service-cart/internal/logging"
	"go.uber.org/zap"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCart(ctx context.Context, userID string) (*CartDocument, error) {
	return s.repo.GetCartByUserID(ctx, userID)
}

func (s *Service) AddItem(ctx context.Context, userID string, item *pb.CartItem) (*CartDocument, error) {
	cart, err := s.repo.GetCartByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	found := false
	for _, existingItem := range cart.Items {
		if existingItem.ProductId == item.ProductId {
			existingItem.Quantity += item.Quantity
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, item)
	}

	err = s.repo.UpdateCart(ctx, cart)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *Service) HandleOrderConfirmedEvent(ctx context.Context, event broker.OrderConfirmedEvent) {
	logging.Info(ctx, "clearing cart for user",
		zap.String("user_id", event.UserID),
		zap.String("order_id", event.OrderID),
	)

	err := s.repo.ClearCart(ctx, event.UserID)
	if err != nil {
		logging.Error(ctx, "failed to clear cart for user", err,
			zap.String("user_id", event.UserID),
		)
	} else {
		logging.Info(ctx, "cart cleared successfully",
			zap.String("user_id", event.UserID),
		)
	}
}
