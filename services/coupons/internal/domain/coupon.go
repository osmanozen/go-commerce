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

type couponTag struct{}
type usageTag struct{}

type CouponID = types.TypedID[couponTag]
type UsageID = types.TypedID[usageTag]

func NewCouponID() CouponID                         { return types.NewTypedID[couponTag]() }
func NewUsageID() UsageID                           { return types.NewTypedID[usageTag]() }
func CouponIDFromString(s string) (CouponID, error) { return types.TypedIDFromString[couponTag](s) }

// ─── Discount Type Enum ──────────────────────────────────────────────────────

type DiscountType int

const (
	DiscountTypeUnknown    DiscountType = iota
	DiscountTypePercentage              // e.g., 10% off
	DiscountTypeFixed                   // e.g., $20 off
	DiscountTypeFreeShipping
)

// ─── Coupon Usage (Owned Entity) ─────────────────────────────────────────────

type CouponUsage struct {
	ID       UsageID   `json:"id" db:"id"`
	CouponID CouponID  `json:"couponId" db:"coupon_id"`
	UserID   string    `json:"userId" db:"user_id"`
	OrderID  uuid.UUID `json:"orderId" db:"order_id"`
	UsedAt   time.Time `json:"usedAt" db:"used_at"`
}

// ─── Coupon Aggregate Root ───────────────────────────────────────────────────

// Coupon is the aggregate root for discount/promotion management.
// Implements multi-level validation: date range, usage limits, minimum order, category restrictions.
type Coupon struct {
	bbdomain.BaseAggregateRoot
	bbdomain.Auditable
	bbdomain.Versionable

	ID               CouponID         `json:"id" db:"id"`
	Code             string           `json:"code" db:"code"`
	Description      string           `json:"description" db:"description"`
	DiscountType     DiscountType     `json:"discountType" db:"discount_type"`
	DiscountValue    decimal.Decimal  `json:"discountValue" db:"discount_value"`
	MinOrderAmount   decimal.Decimal  `json:"minOrderAmount" db:"min_order_amount"`
	MaxDiscount      *decimal.Decimal `json:"maxDiscount,omitempty" db:"max_discount"`
	StartDate        time.Time        `json:"startDate" db:"start_date"`
	EndDate          time.Time        `json:"endDate" db:"end_date"`
	MaxUsages        int              `json:"maxUsages" db:"max_usages"`
	MaxUsagesPerUser int              `json:"maxUsagesPerUser" db:"max_usages_per_user"`
	CategoryIDs      []uuid.UUID      `json:"categoryIds,omitempty"`
	IsActive         bool             `json:"isActive" db:"is_active"`
	Usages           []CouponUsage    `json:"usages,omitempty"`
}

// NewCoupon creates a new validated Coupon aggregate.
func NewCoupon(
	code, description string,
	discountType DiscountType,
	discountValue, minOrderAmount decimal.Decimal,
	maxDiscount *decimal.Decimal,
	startDate, endDate time.Time,
	maxUsages, maxUsagesPerUser int,
) (*Coupon, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) < 3 || len(code) > 50 {
		return nil, errors.New("coupon code must be 3-50 characters")
	}
	if discountType == DiscountTypeUnknown {
		return nil, errors.New("discount type is required")
	}
	if !discountValue.IsPositive() {
		return nil, errors.New("discount value must be positive")
	}
	if endDate.Before(startDate) {
		return nil, errors.New("end date must be after start date")
	}
	if discountType == DiscountTypePercentage && discountValue.GreaterThan(decimal.NewFromInt(100)) {
		return nil, errors.New("percentage discount cannot exceed 100%")
	}

	c := &Coupon{
		ID:               NewCouponID(),
		Code:             code,
		Description:      description,
		DiscountType:     discountType,
		DiscountValue:    discountValue,
		MinOrderAmount:   minOrderAmount,
		MaxDiscount:      maxDiscount,
		StartDate:        startDate,
		EndDate:          endDate,
		MaxUsages:        maxUsages,
		MaxUsagesPerUser: maxUsagesPerUser,
		IsActive:         true,
		Usages:           []CouponUsage{},
	}
	c.SetCreated()
	return c, nil
}

// Validate checks if the coupon can be applied to a given order.
// Multi-level validation: active → date range → usage limits → minimum order → per-user limit.
func (c *Coupon) Validate(userID string, orderAmount decimal.Decimal) error {
	if !c.IsActive {
		return errors.New("coupon is inactive")
	}

	now := time.Now().UTC()
	if now.Before(c.StartDate) {
		return errors.New("coupon is not yet active")
	}
	if now.After(c.EndDate) {
		return errors.New("coupon has expired")
	}

	// Check total usage limit.
	if c.MaxUsages > 0 && len(c.Usages) >= c.MaxUsages {
		return errors.New("coupon usage limit reached")
	}

	// Check minimum order amount.
	if orderAmount.LessThan(c.MinOrderAmount) {
		return fmt.Errorf("order must be at least %s", c.MinOrderAmount.StringFixed(2))
	}

	// Check per-user usage limit.
	if c.MaxUsagesPerUser > 0 {
		userUsages := 0
		for _, usage := range c.Usages {
			if usage.UserID == userID {
				userUsages++
			}
		}
		if userUsages >= c.MaxUsagesPerUser {
			return errors.New("you have reached the maximum usage for this coupon")
		}
	}

	return nil
}

// Apply records a coupon usage and returns the discount amount.
func (c *Coupon) Apply(userID string, orderID uuid.UUID, orderAmount decimal.Decimal) (decimal.Decimal, error) {
	if err := c.Validate(userID, orderAmount); err != nil {
		return decimal.Zero, err
	}

	discount := c.CalculateDiscount(orderAmount)

	c.Usages = append(c.Usages, CouponUsage{
		ID:       NewUsageID(),
		CouponID: c.ID,
		UserID:   userID,
		OrderID:  orderID,
		UsedAt:   time.Now().UTC(),
	})

	c.SetUpdated()
	c.IncrementVersion()

	c.AddDomainEvent(&CouponAppliedEvent{
		BaseDomainEvent: bbdomain.NewBaseDomainEvent(),
		CouponID:        c.ID.Value(),
		Code:            c.Code,
		UserID:          userID,
		OrderID:         orderID,
		Discount:        discount,
	})

	return discount, nil
}

// CalculateDiscount calculates the discount amount without applying it.
func (c *Coupon) CalculateDiscount(orderAmount decimal.Decimal) decimal.Decimal {
	var discount decimal.Decimal

	switch c.DiscountType {
	case DiscountTypePercentage:
		discount = orderAmount.Mul(c.DiscountValue).Div(decimal.NewFromInt(100)).Round(2)
	case DiscountTypeFixed:
		discount = c.DiscountValue
	case DiscountTypeFreeShipping:
		return decimal.Zero // handled separately by shipping calculation
	default:
		return decimal.Zero
	}

	// Cap discount if max discount is set.
	if c.MaxDiscount != nil && discount.GreaterThan(*c.MaxDiscount) {
		discount = *c.MaxDiscount
	}

	// Discount cannot exceed order amount.
	if discount.GreaterThan(orderAmount) {
		discount = orderAmount
	}

	return discount
}

// ─── Domain Events ───────────────────────────────────────────────────────────

type CouponAppliedEvent struct {
	bbdomain.BaseDomainEvent
	CouponID uuid.UUID       `json:"couponId"`
	Code     string          `json:"code"`
	UserID   string          `json:"userId"`
	OrderID  uuid.UUID       `json:"orderId"`
	Discount decimal.Decimal `json:"discount"`
}

func (e *CouponAppliedEvent) EventType() string { return "coupons.coupon.applied" }
