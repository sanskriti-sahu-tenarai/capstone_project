package service

import (
	"context"
	"fmt"
	"log/slog"

	"order-management/model"
	"order-management/repository"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error)
	GetOrder(ctx context.Context, id int64) (*model.Order, error)
	ListOrders(ctx context.Context, limit, offset int) ([]model.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error)
}

type orderService struct {
	repo          repository.OrderRepository
	productClient ProductClient
}

func NewOrderService(repo repository.OrderRepository, pc ProductClient) OrderService {
	return &orderService{
		repo:          repo,
		productClient: pc,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	reqID, _ := ctx.Value("request_id").(string)
	slog.Info("OrderService: querying Product Microservice for price", "request_id", reqID, "product_id", req.ProductID)

	price, err := s.productClient.GetProductPrice(ctx, req.ProductID)
	if err != nil {
		slog.Warn("Product Microservice unreachable. Applying graceful fallback pricing", "request_id", reqID, "err", err)
		price = 50.0 // Fail-safe default price
	}

	order := &model.Order{
		CustomerID: req.CustomerID,
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		TotalPrice: price * float64(req.Quantity),
	}

	return s.repo.Create(ctx, order)
}

func (s *orderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *orderService) ListOrders(ctx context.Context, limit, offset int) ([]model.Order, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
	if !status.IsValid() {
		return nil, model.ErrInvalidStatus
	}

	curr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only block if trying to revert back to CREATED from CANCELLED
	if curr.Status == model.StatusCancelled && status == model.StatusCreated {
		return nil, fmt.Errorf("cannot revert a cancelled order back to created")
	}

	return s.repo.UpdateStatus(ctx, id, status)
}
