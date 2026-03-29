package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/messaging"
	"github.com/osmanozen/oo-commerce/services/reviews/internal/domain"
	"github.com/shopspring/decimal"
)

// ReviewRatingPublisher publishes rating statistics to Kafka
// so the Catalog service can update its product's cached rating.
//
// Flow: Review Created/Updated/Deleted → Recalculate Stats → Kafka → Catalog
type ReviewRatingPublisher struct {
	producer   messaging.EventBus
	reviewRepo domain.ReviewRepository
	logger     *slog.Logger
}

func NewReviewRatingPublisher(producer messaging.EventBus, reviewRepo domain.ReviewRepository, logger *slog.Logger) *ReviewRatingPublisher {
	return &ReviewRatingPublisher{
		producer:   producer,
		reviewRepo: reviewRepo,
		logger:     logger,
	}
}

// PublishRatingUpdate recalculates and publishes rating stats for a product.
func (p *ReviewRatingPublisher) PublishRatingUpdate(ctx context.Context, stats *domain.RatingStats) error {
	type ratingUpdatePayload struct {
		ProductID     string  `json:"productId"`
		AverageRating float64 `json:"averageRating"`
		ReviewCount   int     `json:"reviewCount"`
	}

	avgFloat, _ := stats.AverageRating.Float64()
	payload, err := json.Marshal(ratingUpdatePayload{
		ProductID:     stats.ProductID.String(),
		AverageRating: avgFloat,
		ReviewCount:   stats.ReviewCount,
	})
	if err != nil {
		return fmt.Errorf("serializing rating update: %w", err)
	}

	if err := p.producer.Publish(ctx, "reviews.review.created", stats.ProductID.String(), payload); err != nil {
		return fmt.Errorf("publishing rating update: %w", err)
	}

	p.logger.InfoContext(ctx, "rating update published to catalog",
		slog.String("product_id", stats.ProductID.String()),
		slog.String("avg_rating", stats.AverageRating.StringFixed(2)),
		slog.Int("review_count", stats.ReviewCount),
	)

	return nil
}

// Compile-time unused import guard.
var _ = decimal.Zero
