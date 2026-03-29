module github.com/osmanozen/oo-commerce/services/wishlists

go 1.26.1

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/osmanozen/oo-commerce/pkg/buildingblocks v0.0.0
)

require github.com/shopspring/decimal v1.4.0 // indirect

replace github.com/osmanozen/oo-commerce/pkg/buildingblocks => ../../pkg/buildingblocks
