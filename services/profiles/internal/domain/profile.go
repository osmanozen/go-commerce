package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	bbdomain "github.com/osmanozen/oo-commerce/pkg/buildingblocks/domain"
	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/types"
)

// ─── Strongly-Typed IDs ──────────────────────────────────────────────────────

type profileTag struct{}
type addressTag struct{}

type ProfileID = types.TypedID[profileTag]
type AddressID = types.TypedID[addressTag]

func NewProfileID() ProfileID                         { return types.NewTypedID[profileTag]() }
func NewAddressID() AddressID                         { return types.NewTypedID[addressTag]() }
func ProfileIDFromString(s string) (ProfileID, error) { return types.TypedIDFromString[profileTag](s) }

// ─── Address (Owned Entity) ─────────────────────────────────────────────────

type Address struct {
	ID        AddressID `json:"id" db:"id"`
	ProfileID ProfileID `json:"profileId" db:"profile_id"`
	Label     string    `json:"label" db:"label"`
	FirstName string    `json:"firstName" db:"first_name"`
	LastName  string    `json:"lastName" db:"last_name"`
	Street    string    `json:"street" db:"street"`
	City      string    `json:"city" db:"city"`
	State     string    `json:"state" db:"state"`
	ZipCode   string    `json:"zipCode" db:"zip_code"`
	Country   string    `json:"country" db:"country"`
	Phone     string    `json:"phone" db:"phone"`
	IsDefault bool      `json:"isDefault" db:"is_default"`
}

// ─── User Profile Aggregate Root ─────────────────────────────────────────────

type UserProfile struct {
	bbdomain.BaseAggregateRoot
	bbdomain.Auditable
	bbdomain.Versionable

	ID        ProfileID          `json:"id" db:"id"`
	UserID    string             `json:"userId" db:"user_id"`
	Email     types.Email        `json:"email"`
	FirstName string             `json:"firstName" db:"first_name"`
	LastName  string             `json:"lastName" db:"last_name"`
	Phone     *types.PhoneNumber `json:"phone,omitempty"`
	AvatarURL *string            `json:"avatarUrl,omitempty" db:"avatar_url"`
	Addresses []Address          `json:"addresses,omitempty"`
}

// NewUserProfile creates a new UserProfile aggregate.
func NewUserProfile(userID, email, firstName, lastName string) (*UserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	emailVO, err := types.NewEmail(email)
	if err != nil {
		return nil, err
	}

	p := &UserProfile{
		ID:        NewProfileID(),
		UserID:    userID,
		Email:     emailVO,
		FirstName: strings.TrimSpace(firstName),
		LastName:  strings.TrimSpace(lastName),
		Addresses: []Address{},
	}
	p.SetCreated()

	p.AddDomainEvent(&ProfileCreatedEvent{
		BaseDomainEvent: bbdomain.NewBaseDomainEvent(),
		ProfileID:       p.ID.Value(),
		UserID:          userID,
	})

	return p, nil
}

// AddAddress adds a new address to the profile.
func (p *UserProfile) AddAddress(addr Address) {
	addr.ID = NewAddressID()
	addr.ProfileID = p.ID

	// If this is the first address or marked as default, update default.
	if len(p.Addresses) == 0 || addr.IsDefault {
		for i := range p.Addresses {
			p.Addresses[i].IsDefault = false
		}
		addr.IsDefault = true
	}

	p.Addresses = append(p.Addresses, addr)
	p.SetUpdated()
	p.IncrementVersion()
}

// SetDefaultAddress marks an address as the default.
func (p *UserProfile) SetDefaultAddress(addressID AddressID) error {
	found := false
	for i := range p.Addresses {
		if p.Addresses[i].ID == addressID {
			p.Addresses[i].IsDefault = true
			found = true
		} else {
			p.Addresses[i].IsDefault = false
		}
	}
	if !found {
		return errors.New("address not found")
	}
	p.SetUpdated()
	p.IncrementVersion()
	return nil
}

// ─── Domain Events ───────────────────────────────────────────────────────────

type ProfileCreatedEvent struct {
	bbdomain.BaseDomainEvent
	ProfileID uuid.UUID `json:"profileId"`
	UserID    string    `json:"userId"`
}

func (e *ProfileCreatedEvent) EventType() string { return "profiles.profile.created" }
