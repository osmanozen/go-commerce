package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	bbdomain "github.com/osmanozen/oo-commerce/pkg/buildingblocks/domain"
	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/types"
)

// ─── Strongly-Typed IDs ──────────────────────────────────────────────────────

type wishlistItemTag struct{}

type WishlistItemID = types.TypedID[wishlistItemTag]

func NewWishlistItemID() WishlistItemID { return types.NewTypedID[wishlistItemTag]() }
func WishlistItemIDFromString(s string) (WishlistItemID, error) {
	return types.TypedIDFromString[wishlistItemTag](s)
}

// ─── Wishlist Item (Aggregate Root — simple domain) ─────────────────────────

// WishlistItem is the aggregate root for the wishlists bounded context.
// Simplified domain model — each item is independent (no parent Wishlist entity).
type WishlistItem struct {
	bbdomain.BaseAggregateRoot
	bbdomain.Auditable

	ID        WishlistItemID `json:"id" db:"id"`
	UserID    string         `json:"userId" db:"user_id"`
	ProductID uuid.UUID      `json:"productId" db:"product_id"`
	AddedAt   time.Time      `json:"addedAt" db:"added_at"`
}

// NewWishlistItem creates a new wishlist item.
func NewWishlistItem(userID string, productID uuid.UUID) (*WishlistItem, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	item := &WishlistItem{
		ID:        NewWishlistItemID(),
		UserID:    userID,
		ProductID: productID,
		AddedAt:   time.Now().UTC(),
	}
	item.SetCreated()
	return item, nil
}

// ─── Repository ──────────────────────────────────────────────────────────────

type WishlistRepository interface {
	Add(ctx interface{}, item *WishlistItem) error
	Remove(ctx interface{}, userID string, productID uuid.UUID) error
	GetByUserID(ctx interface{}, userID string, offset, limit int) ([]WishlistItem, int, error)
	Exists(ctx interface{}, userID string, productID uuid.UUID) (bool, error)
	// BatchLookup returns which of the given product IDs the user has wishlisted.
	BatchLookup(ctx interface{}, userID string, productIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}
