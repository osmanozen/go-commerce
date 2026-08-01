package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	bberrors "github.com/osmanozen/go-commerce/src/pkg/buildingblocks/errors"
	"github.com/osmanozen/go-commerce/src/services/coupons/internal/domain"
	"github.com/shopspring/decimal"
)

type PostgresCouponRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCouponRepository(pool *pgxpool.Pool) *PostgresCouponRepository {
	return &PostgresCouponRepository{pool: pool}
}

func (r *PostgresCouponRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO coupons.coupons (
			id, code, description, discount_type, discount_value, min_order_amount,
			max_discount_amount, usage_limit, usage_per_user, times_used, valid_from,
			valid_until, is_active, applicable_product_ids, applicable_category_ids,
			created_at, updated_at, version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		coupon.ID.Value(),
		coupon.Code,
		coupon.Description,
		coupon.DiscountType,
		coupon.DiscountValue,
		coupon.MinOrderAmount,
		coupon.MaxDiscountAmount,
		coupon.UsageLimit,
		coupon.UsagePerUser,
		coupon.TimesUsed,
		coupon.ValidFrom,
		coupon.ValidUntil,
		coupon.IsActive,
		coupon.ApplicableProductIDs,
		coupon.ApplicableCategoryIDs,
		coupon.CreatedAt,
		coupon.UpdatedAt,
		coupon.Version,
	)
	if err != nil {
		return fmt.Errorf("create coupon: %w", err)
	}
	return nil
}

func (r *PostgresCouponRepository) Update(ctx context.Context, coupon *domain.Coupon) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE coupons.coupons
		SET description = $2,
		    discount_type = $3,
		    discount_value = $4,
		    min_order_amount = $5,
		    max_discount_amount = $6,
		    usage_limit = $7,
		    usage_per_user = $8,
		    times_used = $9,
		    valid_from = $10,
		    valid_until = $11,
		    is_active = $12,
		    applicable_product_ids = $13,
		    applicable_category_ids = $14,
		    updated_at = $15,
		    version = $16
		WHERE id = $1 AND version = $17
	`,
		coupon.ID.Value(),
		coupon.Description,
		coupon.DiscountType,
		coupon.DiscountValue,
		coupon.MinOrderAmount,
		coupon.MaxDiscountAmount,
		coupon.UsageLimit,
		coupon.UsagePerUser,
		coupon.TimesUsed,
		coupon.ValidFrom,
		coupon.ValidUntil,
		coupon.IsActive,
		coupon.ApplicableProductIDs,
		coupon.ApplicableCategoryIDs,
		coupon.UpdatedAt,
		coupon.Version,
		coupon.Version-1, // old version
	)
	if err != nil {
		return fmt.Errorf("update coupon: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return bberrors.NewDomainError(bberrors.ErrConcurrencyConflict, "coupon was updated by another process")
	}

	for _, usage := range coupon.Usages {
		_, err = tx.Exec(ctx, `
			INSERT INTO coupons.coupon_usages (id, coupon_id, order_id, user_id, discount_applied, used_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING
		`,
			usage.ID.Value(),
			usage.CouponID.Value(),
			usage.OrderID,
			usage.UserID,
			usage.DiscountApplied,
			usage.UsedAt,
		)
		if err != nil {
			return fmt.Errorf("insert coupon usage: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *PostgresCouponRepository) GetByID(ctx context.Context, id domain.CouponID) (*domain.Coupon, error) {
	var (
		dbID                  uuid.UUID
		code                  string
		description           string
		discountType          int
		discountValue         decimal.Decimal
		minOrderAmount        *decimal.Decimal
		maxDiscountAmount     *decimal.Decimal
		usageLimit            *int
		usagePerUser          *int
		timesUsed             int
		validFrom             time.Time
		validUntil            *time.Time
		isActive              bool
		applicableProductIDs  []uuid.UUID
		applicableCategoryIDs []uuid.UUID
		createdAt             time.Time
		updatedAt             time.Time
		version               int
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, code, description, discount_type, discount_value, min_order_amount,
		       max_discount_amount, usage_limit, usage_per_user, times_used, valid_from,
		       valid_until, is_active, applicable_product_ids, applicable_category_ids,
		       created_at, updated_at, version
		FROM coupons.coupons
		WHERE id = $1
	`, id.Value()).Scan(
		&dbID, &code, &description, &discountType, &discountValue, &minOrderAmount,
		&maxDiscountAmount, &usageLimit, &usagePerUser, &timesUsed, &validFrom,
		&validUntil, &isActive, &applicableProductIDs, &applicableCategoryIDs,
		&createdAt, &updatedAt, &version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select coupon by id: %w", err)
	}

	couponID, err := domain.CouponIDFromString(dbID.String())
	if err != nil {
		return nil, fmt.Errorf("parse coupon id: %w", err)
	}

	coupon := &domain.Coupon{
		ID:                    couponID,
		Code:                  code,
		Description:           description,
		DiscountType:          domain.DiscountType(discountType),
		DiscountValue:         discountValue,
		MinOrderAmount:        minOrderAmount,
		MaxDiscountAmount:     maxDiscountAmount,
		UsageLimit:            usageLimit,
		UsagePerUser:          usagePerUser,
		TimesUsed:             timesUsed,
		ValidFrom:             validFrom,
		ValidUntil:            validUntil,
		IsActive:              isActive,
		ApplicableProductIDs:  applicableProductIDs,
		ApplicableCategoryIDs: applicableCategoryIDs,
	}
	coupon.CreatedAt = createdAt
	coupon.UpdatedAt = updatedAt
	coupon.Version = version
	coupon.ClearDomainEvents()

	usages, err := r.getUsagesByCouponID(ctx, couponID)
	if err != nil {
		return nil, fmt.Errorf("load coupon usages: %w", err)
	}
	coupon.Usages = usages

	return coupon, nil
}

func (r *PostgresCouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	var (
		dbID                  uuid.UUID
		dbCode                string
		description           string
		discountType          int
		discountValue         decimal.Decimal
		minOrderAmount        *decimal.Decimal
		maxDiscountAmount     *decimal.Decimal
		usageLimit            *int
		usagePerUser          *int
		timesUsed             int
		validFrom             time.Time
		validUntil            *time.Time
		isActive              bool
		applicableProductIDs  []uuid.UUID
		applicableCategoryIDs []uuid.UUID
		createdAt             time.Time
		updatedAt             time.Time
		version               int
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, code, description, discount_type, discount_value, min_order_amount,
		       max_discount_amount, usage_limit, usage_per_user, times_used, valid_from,
		       valid_until, is_active, applicable_product_ids, applicable_category_ids,
		       created_at, updated_at, version
		FROM coupons.coupons
		WHERE UPPER(code) = UPPER($1)
	`, strings.TrimSpace(code)).Scan(
		&dbID, &dbCode, &description, &discountType, &discountValue, &minOrderAmount,
		&maxDiscountAmount, &usageLimit, &usagePerUser, &timesUsed, &validFrom,
		&validUntil, &isActive, &applicableProductIDs, &applicableCategoryIDs,
		&createdAt, &updatedAt, &version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select coupon by code: %w", err)
	}

	couponID, err := domain.CouponIDFromString(dbID.String())
	if err != nil {
		return nil, fmt.Errorf("parse coupon id: %w", err)
	}

	coupon := &domain.Coupon{
		ID:                    couponID,
		Code:                  dbCode,
		Description:           description,
		DiscountType:          domain.DiscountType(discountType),
		DiscountValue:         discountValue,
		MinOrderAmount:        minOrderAmount,
		MaxDiscountAmount:     maxDiscountAmount,
		UsageLimit:            usageLimit,
		UsagePerUser:          usagePerUser,
		TimesUsed:             timesUsed,
		ValidFrom:             validFrom,
		ValidUntil:            validUntil,
		IsActive:              isActive,
		ApplicableProductIDs:  applicableProductIDs,
		ApplicableCategoryIDs: applicableCategoryIDs,
	}
	coupon.CreatedAt = createdAt
	coupon.UpdatedAt = updatedAt
	coupon.Version = version
	coupon.ClearDomainEvents()

	usages, err := r.getUsagesByCouponID(ctx, couponID)
	if err != nil {
		return nil, fmt.Errorf("load coupon usages: %w", err)
	}
	coupon.Usages = usages

	return coupon, nil
}

func (r *PostgresCouponRepository) List(ctx context.Context, filter domain.CouponListFilter, offset int, limit int) ([]*domain.Coupon, int, error) {
	whereParts := []string{"1=1"}
	args := make([]interface{}, 0)
	argPos := 1

	if filter.IsActive != nil {
		whereParts = append(whereParts, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *filter.IsActive)
		argPos++
	}

	if filter.Search != "" {
		whereParts = append(whereParts, fmt.Sprintf("(UPPER(code) LIKE UPPER($%d) OR UPPER(description) LIKE UPPER($%d))", argPos, argPos))
		args = append(args, "%"+filter.Search+"%")
		argPos++
	}

	whereClause := strings.Join(whereParts, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM coupons.coupons WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count coupons list: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, code, description, discount_type, discount_value, min_order_amount,
		       max_discount_amount, usage_limit, usage_per_user, times_used, valid_from,
		       valid_until, is_active, applicable_product_ids, applicable_category_ids,
		       created_at, updated_at, version
		FROM coupons.coupons
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	listArgs := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query coupons list: %w", err)
	}
	defer rows.Close()

	var coupons []*domain.Coupon
	for rows.Next() {
		var (
			dbID                  uuid.UUID
			code                  string
			description           string
			discountType          int
			discountValue         decimal.Decimal
			minOrderAmount        *decimal.Decimal
			maxDiscountAmount     *decimal.Decimal
			usageLimit            *int
			usagePerUser          *int
			timesUsed             int
			validFrom             time.Time
			validUntil            *time.Time
			isActive              bool
			applicableProductIDs  []uuid.UUID
			applicableCategoryIDs []uuid.UUID
			createdAt             time.Time
			updatedAt             time.Time
			version               int
		)

		err := rows.Scan(
			&dbID, &code, &description, &discountType, &discountValue, &minOrderAmount,
			&maxDiscountAmount, &usageLimit, &usagePerUser, &timesUsed, &validFrom,
			&validUntil, &isActive, &applicableProductIDs, &applicableCategoryIDs,
			&createdAt, &updatedAt, &version,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan coupon row: %w", err)
		}

		couponID, err := domain.CouponIDFromString(dbID.String())
		if err != nil {
			return nil, 0, fmt.Errorf("parse coupon id: %w", err)
		}

		coupon := &domain.Coupon{
			ID:                    couponID,
			Code:                  code,
			Description:           description,
			DiscountType:          domain.DiscountType(discountType),
			DiscountValue:         discountValue,
			MinOrderAmount:        minOrderAmount,
			MaxDiscountAmount:     maxDiscountAmount,
			UsageLimit:            usageLimit,
			UsagePerUser:          usagePerUser,
			TimesUsed:             timesUsed,
			ValidFrom:             validFrom,
			ValidUntil:            validUntil,
			IsActive:              isActive,
			ApplicableProductIDs:  applicableProductIDs,
			ApplicableCategoryIDs: applicableCategoryIDs,
		}
		coupon.CreatedAt = createdAt
		coupon.UpdatedAt = updatedAt
		coupon.Version = version
		coupon.ClearDomainEvents()

		coupons = append(coupons, coupon)
	}

	return coupons, total, nil
}

func (r *PostgresCouponRepository) CountUserUsage(ctx context.Context, couponID domain.CouponID, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM coupons.coupon_usages
		WHERE coupon_id = $1 AND user_id = $2
	`, couponID.Value(), strings.TrimSpace(userID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user coupon usages: %w", err)
	}
	return count, nil
}

func (r *PostgresCouponRepository) getUsagesByCouponID(ctx context.Context, couponID domain.CouponID) ([]domain.CouponUsage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, coupon_id, order_id, user_id, discount_applied, used_at
		FROM coupons.coupon_usages
		WHERE coupon_id = $1
	`, couponID.Value())
	if err != nil {
		return nil, fmt.Errorf("query coupon usages: %w", err)
	}
	defer rows.Close()

	var usages []domain.CouponUsage
	for rows.Next() {
		var (
			dbID            uuid.UUID
			dbCouponID      uuid.UUID
			orderID         uuid.UUID
			userID          string
			discountApplied decimal.Decimal
			usedAt          time.Time
		)
		err := rows.Scan(&dbID, &dbCouponID, &orderID, &userID, &discountApplied, &usedAt)
		if err != nil {
			return nil, fmt.Errorf("scan coupon usage: %w", err)
		}

		usageID, err := domain.CouponUsageIDFromString(dbID.String())
		if err != nil {
			return nil, fmt.Errorf("parse coupon usage id: %w", err)
		}

		usages = append(usages, domain.CouponUsage{
			ID:              usageID,
			CouponID:        couponID,
			OrderID:         orderID,
			UserID:          userID,
			DiscountApplied: discountApplied,
			UsedAt:          usedAt,
		})
	}
	return usages, nil
}

var _ domain.CouponRepository = (*PostgresCouponRepository)(nil)
