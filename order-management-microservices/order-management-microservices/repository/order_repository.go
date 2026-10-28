package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"order-management/model"
	"order-management/observability"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) (*model.Order, error)
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	List(ctx context.Context, limit, offset int) ([]model.Order, error)
	UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error)
}

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *model.Order) (*model.Order, error) {
	query := `
		INSERT INTO orders (customer_id, product_id, quantity, total_price, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	now := time.Now().UTC()
	err := r.db.QueryRowContext(ctx, query,
		order.CustomerID,
		order.ProductID,
		order.Quantity,
		order.TotalPrice,
		model.StatusCreated,
		now,
		now,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		observability.DBErrorCount.Inc()
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}
	order.Status = model.StatusCreated
	return order, nil
}

func (r *orderRepository) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	query := `
		SELECT id, customer_id, product_id, quantity, total_price, status, created_at, updated_at
		FROM orders
		WHERE id = $1`

	var o model.Order
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		observability.DBErrorCount.Inc()
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &o, nil
}

func (r *orderRepository) List(ctx context.Context, limit, offset int) ([]model.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, customer_id, product_id, quantity, total_price, status, created_at, updated_at
		FROM orders
		ORDER BY id DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		observability.DBErrorCount.Inc()
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			observability.DBErrorCount.Inc()
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, customer_id, product_id, quantity, total_price, status, created_at, updated_at`

	var o model.Order
	err := r.db.QueryRowContext(ctx, query, status, time.Now().UTC(), id).Scan(
		&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		observability.DBErrorCount.Inc()
		return nil, fmt.Errorf("failed to update status: %w", err)
	}
	return &o, nil
}
