-- Coupons Service Database Schema
-- Schema: coupons

CREATE SCHEMA IF NOT EXISTS coupons;

-- Coupons table
CREATE TABLE coupons.coupons (
    id uuid PRIMARY KEY,
    code varchar(50) NOT NULL UNIQUE,
    description varchar(500) NOT NULL DEFAULT '',
    discount_type int NOT NULL,
    discount_value numeric(18,2) NOT NULL CHECK (discount_value > 0),
    min_order_amount numeric(18,2),
    max_discount_amount numeric(18,2),
    usage_limit int,
    usage_per_user int,
    times_used int NOT NULL DEFAULT 0,
    valid_from timestamptz NOT NULL,
    valid_until timestamptz,
    is_active boolean NOT NULL DEFAULT true,
    applicable_product_ids uuid[] NOT NULL DEFAULT '{}',
    applicable_category_ids uuid[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version int NOT NULL DEFAULT 0
);

CREATE INDEX idx_coupons_code ON coupons.coupons(code);
CREATE INDEX idx_coupons_active ON coupons.coupons(is_active, valid_from, valid_until);

-- Coupon usage tracking
CREATE TABLE coupons.coupon_usages (
    id uuid PRIMARY KEY,
    coupon_id uuid NOT NULL REFERENCES coupons.coupons(id) ON DELETE CASCADE,
    order_id uuid NOT NULL,
    user_id varchar(100) NOT NULL,
    discount_applied numeric(18,2) NOT NULL,
    used_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_usages_coupon_id ON coupons.coupon_usages(coupon_id);
CREATE INDEX idx_usages_user_id ON coupons.coupon_usages(coupon_id, user_id);
CREATE INDEX idx_usages_order_id ON coupons.coupon_usages(order_id);
