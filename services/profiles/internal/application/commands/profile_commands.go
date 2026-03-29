package commands

import (
	"context"

	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/cqrs"
)

type UpdateProfileCommand struct {
	DisplayName string `json:"displayName"`
}

func (c UpdateProfileCommand) CommandName() string { return "UpdateProfileCommand" }

type UpdateProfileHandler struct{}

func NewUpdateProfileHandler() *UpdateProfileHandler {
	return &UpdateProfileHandler{}
}

func (h *UpdateProfileHandler) Handle(ctx context.Context, cmd UpdateProfileCommand) (struct{}, error) {
	// TODO: implement
	return struct{}{}, nil
}

type AddAddressCommand struct {
	Name         string `json:"name"`
	Street       string `json:"street"`
	City         string `json:"city"`
	State        string `json:"state"`
	ZipCode      string `json:"zipCode"`
	Country      string `json:"country"`
	SetAsDefault bool   `json:"setAsDefault"`
}

func (c AddAddressCommand) CommandName() string { return "AddAddressCommand" }

type AddAddressHandler struct{}

func NewAddAddressHandler() *AddAddressHandler {
	return &AddAddressHandler{}
}

func (h *AddAddressHandler) Handle(ctx context.Context, cmd AddAddressCommand) (string, error) {
	// TODO: implement
	return "addr-1", nil
}

type UpdateAddressCommand struct {
	AddressID string `json:"-"`
	Name      string `json:"name"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	ZipCode   string `json:"zipCode"`
	Country   string `json:"country"`
}

func (c UpdateAddressCommand) CommandName() string { return "UpdateAddressCommand" }

type UpdateAddressHandler struct{}

func NewUpdateAddressHandler() *UpdateAddressHandler {
	return &UpdateAddressHandler{}
}

func (h *UpdateAddressHandler) Handle(ctx context.Context, cmd UpdateAddressCommand) (struct{}, error) {
	// TODO: implement
	return struct{}{}, nil
}

type DeleteAddressCommand struct {
	AddressID string `json:"-"`
}

func (c DeleteAddressCommand) CommandName() string { return "DeleteAddressCommand" }

type DeleteAddressHandler struct{}

func NewDeleteAddressHandler() *DeleteAddressHandler {
	return &DeleteAddressHandler{}
}

func (h *DeleteAddressHandler) Handle(ctx context.Context, cmd DeleteAddressCommand) (struct{}, error) {
	// TODO: implement
	return struct{}{}, nil
}

type SetDefaultAddressCommand struct {
	AddressID string `json:"-"`
}

func (c SetDefaultAddressCommand) CommandName() string { return "SetDefaultAddressCommand" }

type SetDefaultAddressHandler struct{}

func NewSetDefaultAddressHandler() *SetDefaultAddressHandler {
	return &SetDefaultAddressHandler{}
}

func (h *SetDefaultAddressHandler) Handle(ctx context.Context, cmd SetDefaultAddressCommand) (struct{}, error) {
	// TODO: implement
	return struct{}{}, nil
}

var (
	_ cqrs.CommandHandler[UpdateProfileCommand, struct{}]     = (*UpdateProfileHandler)(nil)
	_ cqrs.CommandHandler[AddAddressCommand, string]          = (*AddAddressHandler)(nil)
	_ cqrs.CommandHandler[UpdateAddressCommand, struct{}]     = (*UpdateAddressHandler)(nil)
	_ cqrs.CommandHandler[DeleteAddressCommand, struct{}]     = (*DeleteAddressHandler)(nil)
	_ cqrs.CommandHandler[SetDefaultAddressCommand, struct{}] = (*SetDefaultAddressHandler)(nil)
)
