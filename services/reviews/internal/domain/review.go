package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	bbdomain "github.com/osmanozen/oo-commerce/pkg/buildingblocks/domain"
	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/types"
	"github.com/shopspring/decimal"
)

// ─── Strongly-Typed IDs ──────────────────────────────────────────────────────

type reviewTag struct{}

type ReviewID = types.TypedID[reviewTag]

func NewReviewID() ReviewID                         { return types.NewTypedID[reviewTag]() }
func ReviewIDFromString(s string) (ReviewID, error) { return types.TypedIDFromString[reviewTag](s) }

// ─── Value Objects ───────────────────────────────────────────────────────────

// Rating is a validated review rating (1-5 stars).
type Rating struct {
	value int
}

func NewRating(value int) (Rating, error) {
	if value < 1 || value > 5 {
		return Rating{}, errors.New("rating must be between 1 and 5")
	}
	return Rating{value: value}, nil
}

func (r Rating) Value() int     { return r.value }
func (r Rating) String() string { return fmt.Sprintf("%d/5", r.value) }

// ReviewText is a validated review body (10-2000 chars).
type ReviewText struct {
	value string
}

func NewReviewText(text string) (ReviewText, error) {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 10 {
		return ReviewText{}, errors.New("review text must be at least 10 characters")
	}
	if len(trimmed) > 2000 {
		return ReviewText{}, errors.New("review text must be at most 2000 characters")
	}
	return ReviewText{value: trimmed}, nil
}

func (r ReviewText) String() string { return r.value }

// ─── Review Aggregate Root ───────────────────────────────────────────────────

// Review represents a product review with verified purchase tracking.
type Review struct {
	bbdomain.BaseAggregateRoot
	bbdomain.Auditable
	bbdomain.Versionable

	ID              ReviewID   `json:"id" db:"id"`
	ProductID       uuid.UUID  `json:"productId" db:"product_id"`
	UserID          string     `json:"userId" db:"user_id"`
	UserDisplayName string     `json:"userDisplayName" db:"user_display_name"`
	Rating          Rating     `json:"rating"`
	Title           string     `json:"title" db:"title"`
	Body            ReviewText `json:"body"`
	IsVerified      bool       `json:"isVerified" db:"is_verified"`
	HelpfulCount    int        `json:"helpfulCount" db:"helpful_count"`
}

// NewReview creates a new Review with verified purchase validation.
func NewReview(productID uuid.UUID, userID, displayName string, rating int, title, body string, isVerified bool) (*Review, error) {
	ratingVO, err := NewRating(rating)
	if err != nil {
		return nil, err
	}
	bodyVO, err := NewReviewText(body)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(title) == "" || len(title) > 200 {
		return nil, errors.New("title must be 1-200 characters")
	}

	r := &Review{
		ID:              NewReviewID(),
		ProductID:       productID,
		UserID:          userID,
		UserDisplayName: displayName,
		Rating:          ratingVO,
		Title:           strings.TrimSpace(title),
		Body:            bodyVO,
		IsVerified:      isVerified,
	}
	r.SetCreated()

	r.AddDomainEvent(&ReviewCreatedEvent{
		BaseDomainEvent: bbdomain.NewBaseDomainEvent(),
		ReviewID:        r.ID.Value(),
		ProductID:       productID,
		Rating:          rating,
	})

	return r, nil
}

// Update modifies a review's content.
func (r *Review) Update(rating int, title, body string) error {
	ratingVO, err := NewRating(rating)
	if err != nil {
		return err
	}
	bodyVO, err := NewReviewText(body)
	if err != nil {
		return err
	}

	oldRating := r.Rating.Value()
	r.Rating = ratingVO
	r.Title = strings.TrimSpace(title)
	r.Body = bodyVO
	r.SetUpdated()
	r.IncrementVersion()

	r.AddDomainEvent(&ReviewUpdatedEvent{
		BaseDomainEvent: bbdomain.NewBaseDomainEvent(),
		ReviewID:        r.ID.Value(),
		ProductID:       r.ProductID,
		OldRating:       oldRating,
		NewRating:       rating,
	})

	return nil
}

// ─── Rating Statistics ───────────────────────────────────────────────────────

// RatingStats holds aggregated rating statistics for a product.
type RatingStats struct {
	ProductID     uuid.UUID       `json:"productId" db:"product_id"`
	AverageRating decimal.Decimal `json:"averageRating" db:"average_rating"`
	ReviewCount   int             `json:"reviewCount" db:"review_count"`
	Distribution  map[int]int     `json:"distribution"` // star → count
}

// ─── Domain Events ───────────────────────────────────────────────────────────

type ReviewCreatedEvent struct {
	bbdomain.BaseDomainEvent
	ReviewID  uuid.UUID `json:"reviewId"`
	ProductID uuid.UUID `json:"productId"`
	Rating    int       `json:"rating"`
}

func (e *ReviewCreatedEvent) EventType() string { return "reviews.review.created" }

type ReviewUpdatedEvent struct {
	bbdomain.BaseDomainEvent
	ReviewID  uuid.UUID `json:"reviewId"`
	ProductID uuid.UUID `json:"productId"`
	OldRating int       `json:"oldRating"`
	NewRating int       `json:"newRating"`
}

func (e *ReviewUpdatedEvent) EventType() string { return "reviews.review.updated" }

type ReviewDeletedEvent struct {
	bbdomain.BaseDomainEvent
	ReviewID  uuid.UUID `json:"reviewId"`
	ProductID uuid.UUID `json:"productId"`
	Rating    int       `json:"rating"`
}

func (e *ReviewDeletedEvent) EventType() string { return "reviews.review.deleted" }

// ─── Verified Purchase Checker ───────────────────────────────────────────────

// PurchaseVerifier checks if a user has purchased a product (cross-service query).
type PurchaseVerifier interface {
	// HasPurchased returns true if the user has a confirmed order containing the product.
	HasPurchased(ctx interface{}, userID string, productID uuid.UUID) (bool, error)
}

// ─── Review Repository ──────────────────────────────────────────────────────

type ReviewRepository interface {
	Create(ctx interface{}, review *Review) error
	GetByID(ctx interface{}, id ReviewID) (*Review, error)
	GetByProductID(ctx interface{}, productID uuid.UUID, offset, limit int) ([]Review, int, error)
	Update(ctx interface{}, review *Review) error
	Delete(ctx interface{}, id ReviewID) error
	GetRatingStats(ctx interface{}, productID uuid.UUID) (*RatingStats, error)
	ExistsByUserAndProduct(ctx interface{}, userID string, productID uuid.UUID) (bool, error)
}

// Since review events update the Catalog service's product rating,
// the Reviews consumer publishes rating recalculation events to Kafka.
// The Catalog service subscribes to these events and updates its product cache.
//
// Flow: Review Created → Kafka → Catalog Service → UpdateReviewStats
// This keeps the Catalog's rating stats eventually consistent with Reviews.
type ReviewEventPublisher interface {
	PublishRatingUpdate(ctx interface{}, stats *RatingStats) error
}

type unusedTime = time.Time
