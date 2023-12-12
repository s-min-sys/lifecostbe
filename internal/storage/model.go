package storage

import (
	"github.com/s-min-sys/lifecostbe/internal/model"
)

type Organization struct {
	Persons     map[uint64]model.Person
	Groups      map[uint64]model.Group
	SubWallets  map[uint64]model.Wallet
	Labels      map[uint64]model.Label
	GroupLabels map[uint64]map[uint64]model.Label

	Merchants      map[uint64]model.CostDir
	GroupMerchants map[uint64]map[uint64]model.CostDir
}

func NewOrganization() *Organization {
	organization := &Organization{}

	organization.valid()

	return organization
}

func (organization *Organization) reset() {
	organization.Persons = nil
	organization.Groups = nil
	organization.SubWallets = nil
	organization.Labels = nil
	organization.GroupLabels = nil
	organization.Merchants = nil
	organization.GroupMerchants = nil

	organization.valid()
}

func (organization *Organization) valid() {
	if organization.Persons == nil {
		organization.Persons = make(map[uint64]model.Person)
	}

	if organization.Groups == nil {
		organization.Groups = make(map[uint64]model.Group)
	}

	if organization.SubWallets == nil {
		organization.SubWallets = make(map[uint64]model.Wallet)
	}

	if organization.Labels == nil {
		organization.Labels = make(map[uint64]model.Label)
	}

	if organization.GroupLabels == nil {
		organization.GroupLabels = make(map[uint64]map[uint64]model.Label)
	}

	if organization.Merchants == nil {
		organization.Merchants = make(map[uint64]model.CostDir)
	}

	if organization.GroupMerchants == nil {
		organization.GroupMerchants = make(map[uint64]map[uint64]model.CostDir)
	}
}

type GroupEnterInfo struct {
	PersonID uint64
	GroupID  uint64
}
