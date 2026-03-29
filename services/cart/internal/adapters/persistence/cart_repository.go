package persistence

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osmanozen/oo-commerce/services/cart/internal/domain"
)

type CartRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewCartRepository(pool *pgxpool.Pool, logger *slog.Logger) *CartRepository {
	return &CartRepository{pool: pool, logger: logger}
}

func (r *CartRepository) GetByID(ctx context.Context, id domain.CartID) (*domain.Cart, error) {
	// TODO: implement with pgx
	return nil, nil
}

func (r *CartRepository) GetByUserID(ctx context.Context, userID string) (*domain.Cart, error) {
	// TODO: implement with pgx
	return nil, nil
}

func (r *CartRepository) GetByGuestID(ctx context.Context, guestID string) (*domain.Cart, error) {
	// TODO: implement with pgx
	return nil, nil
}

func (r *CartRepository) Create(ctx context.Context, cart *domain.Cart) error {
	// TODO: implement with pgx
	return nil
}

func (r *CartRepository) Update(ctx context.Context, cart *domain.Cart) error {
	// TODO: implement with pgx
	return nil
}

func (r *CartRepository) Delete(ctx context.Context, id domain.CartID) error {
	// TODO: implement with pgx
	return nil
}

func (r *CartRepository) CleanupAbandoned(ctx context.Context, olderThanHours int) (int64, error) {
	// TODO: implement with pgx
	return 0, nil
}

var _ domain.CartRepository = (*CartRepository)(nil)
