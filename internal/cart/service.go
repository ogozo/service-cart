package cart

import (
	pb "github.com/ogozo/proto-definitions/gen/go/cart"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCart(userID string) (*CartDocument, error) {
	return s.repo.GetCartByUserID(userID)
}

func (s *Service) AddItem(userID string, item *pb.CartItem) (*CartDocument, error) {
	cart, err := s.repo.GetCartByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Ürün zaten sepette var mı diye kontrol et
	found := false
	for _, existingItem := range cart.Items {
		if existingItem.ProductId == item.ProductId {
			existingItem.Quantity += item.Quantity // Miktarı artır
			found = true
			break
		}
	}

	// Eğer ürün sepette yoksa, yeni olarak ekle
	if !found {
		cart.Items = append(cart.Items, item)
	}

	err = s.repo.UpdateCart(userID, cart)
	if err != nil {
		return nil, err
	}
	return cart, nil
}
