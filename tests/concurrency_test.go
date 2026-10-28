package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"order-management/model"
)

type safeOrderRepo struct {
	mu     sync.RWMutex
	orders map[int64]model.Order
	seq    int64
}

func newSafeOrderRepo() *safeOrderRepo {
	return &safeOrderRepo{
		orders: make(map[int64]model.Order),
	}
}

func (m *safeOrderRepo) Create(ctx context.Context, order *model.Order) (*model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	order.ID = m.seq
	order.Status = model.StatusCreated
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	// Store copy to eliminate reference race conditions
	m.orders[order.ID] = *order
	return order, nil
}

func (m *safeOrderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if o, ok := m.orders[id]; ok {
		res := o
		return &res, nil
	}
	return nil, model.ErrOrderNotFound
}

func (m *safeOrderRepo) List(ctx context.Context, limit, offset int) ([]model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]model.Order, 0, len(m.orders))
	for _, o := range m.orders {
		list = append(list, o)
	}
	return list, nil
}

func (m *safeOrderRepo) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if o, ok := m.orders[id]; ok {
		o.Status = status
		o.UpdatedAt = time.Now()
		m.orders[id] = o
		res := o
		return &res, nil
	}
	return nil, model.ErrOrderNotFound
}

func Test100ConcurrentOrders(t *testing.T) {
	repo := newSafeOrderRepo()
	var wg sync.WaitGroup
	var successCount int64
	numRequests := 100

	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			order := &model.Order{
				CustomerID: int64(100 + idx),
				ProductID:  501,
				Quantity:   2,
				TotalPrice: 240.0,
			}
			_, err := repo.Create(context.Background(), order)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}
	wg.Wait()

	if successCount != int64(numRequests) {
		t.Fatalf("Expected %d successful creations, got %d", numRequests, successCount)
	}
}
