package cart

import (
	"context"

	pb "github.com/ogozo/proto-definitions/gen/go/cart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedCartServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AddItemToCart(ctx context.Context, req *pb.AddItemToCartRequest) (*pb.AddItemToCartResponse, error) {
	// Gelen isteği alıp service katmanına iletiyoruz.
	cart, err := h.service.AddItem(req.UserId, req.Item)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not add item to cart: %v", err)
	}

	// Service katmanından dönen güncel sepeti response olarak döndürüyoruz.
	return &pb.AddItemToCartResponse{UserId: cart.UserID, Items: cart.Items}, nil
}

func (h *Handler) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.GetCartResponse, error) {
	cart, err := h.service.GetCart(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get cart: %v", err)
	}
	return &pb.GetCartResponse{UserId: cart.UserID, Items: cart.Items}, nil
}
