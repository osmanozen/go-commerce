package queries

import (
	"context"

	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/cqrs"
)

type GetProfileQuery struct{}

func (q GetProfileQuery) QueryName() string { return "GetProfileQuery" }

type ProfileDTO struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId"`
	DisplayName string       `json:"displayName"`
	AvatarURL   *string      `json:"avatarUrl"`
	Addresses   []AddressDTO `json:"addresses"`
}

type AddressDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	ZipCode   string `json:"zipCode"`
	Country   string `json:"country"`
	IsDefault bool   `json:"isDefault"`
}

type GetProfileHandler struct{}

func NewGetProfileHandler() *GetProfileHandler {
	return &GetProfileHandler{}
}

func (h *GetProfileHandler) Handle(ctx context.Context, query GetProfileQuery) (*ProfileDTO, error) {
	// TODO: implement
	return &ProfileDTO{
		ID:          "prof-1",
		UserID:      "user-1",
		DisplayName: "User",
		Addresses:   []AddressDTO{},
	}, nil
}

var _ cqrs.QueryHandler[GetProfileQuery, *ProfileDTO] = (*GetProfileHandler)(nil)
