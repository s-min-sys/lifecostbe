package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/godruoyi/go-snowflake"
	"github.com/patrickmn/go-cache"
	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/pathutils"
	"github.com/sgostarter/libeasygo/stg/fs/rawfs"
	"github.com/sgostarter/libeasygo/stg/mwf"
	"golang.org/x/exp/slices"
)

const (
	maxTmpDataDuration = time.Hour * 24 * 7
)

type MerchantPersonInfo struct {
	PersonID uint64
	CostDir  model.CostDir
}

type Storage interface {
	NewPerson(name string) (personID, defaultWalletID uint64, err error)
	NewPersonEx(name string, suggestPersonID uint64) (personID, defaultWalletID uint64, err error)
	GetPersonName(personID uint64) (name string, err error)
	GetPersonGroupsIDs(personID uint64) (groupIDs []uint64, err error)
	GetPersonWalletIDs(personID uint64) (subWalletIDs []uint64, err error)
	SetPersonMerchant(personID uint64, costDir model.CostDir) error
	SetPersonGroupMerchant(personID, groupID uint64, costDir model.CostDir) error
	GetMerchantPersons() (merchants []MerchantPersonInfo)
	IsMerchantPerson(personID uint64) (dir model.CostDir, ok bool)
	GetGroupMerchantPersons(groupID uint64) (merchants []MerchantPersonInfo)
	IsGroupMerchantPerson(personID, groupID uint64) (dir model.CostDir, ok bool)

	NewGroup(name string, personID uint64) (id uint64, err error)
	JoinGroup(groupID, personID uint64) error
	LeaveGroup(groupID, personID uint64) error
	SetGroupAdmin(groupID, personID uint64, adminFlag bool) error
	IsGroupAdmin(groupID, personID uint64) (adminFlag bool, err error)
	GetGroupPersonIDs(groupID uint64) (personIDs, adminIDs []uint64, err error)
	GetGroupNames(groupIDs []uint64) (names []string, err error)

	NewWallet(name string, personID uint64) (id uint64, err error)
	GetWallet(walletID uint64) (wallet model.Wallet, err error)

	NewLabel(name string) (id uint64, err error)
	GetLabels() (labels []model.Label, err error)
	GetLabelName(id uint64) (name string, err error)

	NewGroupLabel(groupID uint64, name string) (id uint64, err error)
	GetGroupLabels(groupID uint64) (labels []model.Label, err error)
	GetGroupLabelName(labelID, groupID uint64) (name string, err error)

	Record(groupID uint64, groupBill model.GroupBill) error
	GetBill(groupID uint64, billID string) (bill model.GroupBill, err error)
	DeleteRecord(groupID uint64, recordID string) error
	GetBills(groupID uint64) ([]model.GroupBill, error)
	GetBillsEx(groupID uint64, startYear, startMonth, startDay, finishYear,
		finishMonth, finishDay int) ([]model.GroupBill, error)
	GetBillsByID(groupID uint64, id string, count int, dirNew bool) (bills []model.GroupBill, hasMore bool, err error)

	GetDeletedBills(groupID uint64) (bills []model.DeletedGroupBill, err error)
	GetDeletedBill(groupID uint64, billID string) (bill model.DeletedGroupBill, err error)
	CleanDeletedBill(groupID uint64, billID string) (err error)
	RestoreDeletedBill(groupID uint64, billID string) (err error)

	AddGroupEnterCodes(enterCodes []string, personID, groupID uint64, duration time.Duration) (err error)
	ActiveGroupEnterCode(enterCode string) (personID, groupID uint64, ok bool, err error)
}

func NewStorage(dataRoot string, debug bool, logger l.Wrapper) Storage {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	_ = pathutils.MustDirExists(filepath.Join(dataRoot, "bills"))

	impl := &storageImpl{
		logger:      logger.WithFields(l.StringField(l.ClsKey, "storageImpl")),
		dataRoot:    dataRoot,
		billsRoot:   filepath.Join(dataRoot, "bills"),
		tmpDataFile: filepath.Join(dataRoot, "etd"),
		organization: mwf.NewMemWithFile[*Organization, mwf.Serial, mwf.Lock](
			NewOrganization(), &mwf.JSONSerial{
				MarshalIndent: debug,
			}, &sync.RWMutex{}, "organization", rawfs.NewFSStorage(dataRoot)),
		tmpData:    cache.New(maxTmpDataDuration, maxTmpDataDuration),
		groupBills: make(map[uint64]BillFile),
	}

	impl.init()

	return impl
}

type storageImpl struct {
	logger       l.Wrapper
	organization *mwf.MemWithFile[*Organization, mwf.Serial, mwf.Lock]
	tmpData      *cache.Cache

	dataRoot       string
	billsRoot      string
	tmpDataFile    string
	groupBillsLock sync.Mutex
	groupBills     map[uint64]BillFile
}

func (impl *storageImpl) init() {
	var hasData bool

	_ = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		newOrg.valid()

		hasData = len(newOrg.Labels) > 0

		if !hasData {
			newOrg.reset()
		}

		return
	})

	if !hasData {
		_ = impl.initData()
	}

	_ = impl.tmpData.LoadFile(impl.tmpDataFile)
}

func (impl *storageImpl) getGroupBills(groupID uint64) BillFile {
	impl.groupBillsLock.Lock()
	defer impl.groupBillsLock.Unlock()

	groupBill, ok := impl.groupBills[groupID]
	if !ok {
		groupBill = NewBillFile(groupID, impl.billsRoot, strconv.FormatUint(groupID, 10), impl.logger)

		impl.groupBills[groupID] = groupBill
	}

	return groupBill
}

func (impl *storageImpl) NewPerson(name string) (personID, defaultWalletID uint64, err error) {
	return impl.NewPersonEx(name, 0)
}

func (impl *storageImpl) NewPersonEx(name string, suggestPersonID uint64) (personID, defaultWalletID uint64, err error) {
	err = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		for _, person := range newOrg.Persons {
			if person.ID == suggestPersonID {
				err = commerr.ErrAlreadyExists

				return
			}

			if person.Name == name {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		personID = suggestPersonID
		if personID == 0 {
			personID = snowflake.ID()
		}

		walletIDs := make([]uint64, 0, 5)

		defaultWalletID = personID
		walletIDs = append(walletIDs, defaultWalletID)

		newOrg.SubWallets[defaultWalletID] = model.Wallet{
			ID:       defaultWalletID,
			Name:     "*",
			PersonID: personID,
		}

		if suggestPersonID > 0 { // not Merchant user
			swIDAlipay := snowflake.ID()
			walletIDs = append(walletIDs, swIDAlipay)

			newOrg.SubWallets[swIDAlipay] = model.Wallet{
				ID:       swIDAlipay,
				Name:     "支付宝",
				PersonID: personID,
			}

			swIDWeChat := snowflake.ID()
			walletIDs = append(walletIDs, swIDWeChat)

			newOrg.SubWallets[swIDWeChat] = model.Wallet{
				ID:       swIDWeChat,
				Name:     "微信",
				PersonID: personID,
			}

			swIDBank := snowflake.ID()
			walletIDs = append(walletIDs, swIDBank)

			newOrg.SubWallets[swIDBank] = model.Wallet{
				ID:       swIDBank,
				Name:     "银行账户",
				PersonID: personID,
			}
		}

		newOrg.Persons[personID] = model.Person{
			ID:           personID,
			Name:         name,
			Groups:       nil,
			SubWalletIDs: walletIDs,
		}

		return
	})

	return
}

func (impl *storageImpl) GetPersonName(personID uint64) (name string, err error) {
	impl.organization.Read(func(org *Organization) {
		person, ok := org.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		name = person.Name
	})

	return
}

func (impl *storageImpl) GetPersonGroupsIDs(personID uint64) (groupIDs []uint64, err error) {
	impl.organization.Read(func(org *Organization) {
		person, ok := org.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		groupIDs = person.Groups
	})

	return
}

func (impl *storageImpl) NewGroup(name string, personID uint64) (id uint64, err error) {
	err = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		person, ok := newOrg.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		for _, group := range newOrg.Groups {
			if group.Name == name {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		id = snowflake.ID()

		newOrg.Groups[id] = model.Group{
			ID:   id,
			Name: name,
			MemberPersonIDs: []uint64{
				personID,
			},
			AdminPersonIDs: []uint64{
				personID,
			},
		}

		person.Groups = append(person.Groups, id)
		newOrg.Persons[personID] = person
		/*
			//
			//
			//

			groupMerchantPersonID := snowflake.ID()
			groupMerchantPersonWalletID := groupMerchantPersonID

			newOrg.SubWallets[groupMerchantPersonWalletID] = model.Wallet{
				ID:       groupMerchantPersonWalletID,
				Name:     "*",
				PersonID: groupMerchantPersonID,
			}

			newOrg.Persons[groupMerchantPersonID] = model.Person{
				ID:           groupMerchantPersonID,
				Name:         "进账",
				SubWalletIDs: []uint64{groupMerchantPersonWalletID},
			}

			if newOrg.GroupMerchants[id] == nil {
				newOrg.GroupMerchants[id] = make(map[uint64]model.CostDir)
			}

			newOrg.GroupMerchants[id][groupMerchantPersonID] = model.CostDirOut

			//
			//
			//

			groupMerchantPersonID = snowflake.ID()
			groupMerchantPersonWalletID = groupMerchantPersonID

			newOrg.SubWallets[groupMerchantPersonWalletID] = model.Wallet{
				ID:       groupMerchantPersonWalletID,
				Name:     "*",
				PersonID: groupMerchantPersonID,
			}

			newOrg.Persons[groupMerchantPersonID] = model.Person{
				ID:           groupMerchantPersonID,
				Name:         "消费",
				SubWalletIDs: []uint64{groupMerchantPersonWalletID},
			}

			newOrg.GroupMerchants[id][groupMerchantPersonID] = model.CostDirIn
		*/
		return
	})

	return
}

func (impl *storageImpl) JoinGroup(groupID, personID uint64) error {
	return impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		person, ok := newOrg.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		group, ok := newOrg.Groups[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		for _, memberID := range group.MemberPersonIDs {
			if memberID == personID {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		group.MemberPersonIDs = append(group.MemberPersonIDs, personID)

		newOrg.Groups[groupID] = group

		person.Groups = append(person.Groups, groupID)

		newOrg.Persons[personID] = person

		return
	})
}

func (impl *storageImpl) LeaveGroup(groupID, personID uint64) error {
	return impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		person, ok := newOrg.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		group, ok := newOrg.Groups[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		var memberExists bool

		for idx, memberID := range group.MemberPersonIDs {
			if memberID == personID {
				group.MemberPersonIDs = slices.Delete(group.MemberPersonIDs, idx, idx+1)
				memberExists = true

				break
			}
		}

		if !memberExists {
			err = commerr.ErrNotFound

			return
		}

		for idx, memberID := range group.AdminPersonIDs {
			if memberID == personID {
				group.AdminPersonIDs = slices.Delete(group.AdminPersonIDs, idx, idx+1)

				break
			}
		}

		newOrg.Groups[groupID] = group

		for idx, id := range person.Groups {
			if id == groupID {
				person.Groups = slices.Delete(person.Groups, idx, idx+1)

				break
			}
		}

		return
	})
}

func (impl *storageImpl) GetGroupPersonIDs(groupID uint64) (personIDs, adminIDs []uint64, err error) {
	impl.organization.Read(func(org *Organization) {
		group, ok := org.Groups[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		personIDs = group.MemberPersonIDs
		adminIDs = group.AdminPersonIDs
	})

	return
}

func (impl *storageImpl) GetGroupNames(groupIDs []uint64) (names []string, err error) {
	impl.organization.Read(func(org *Organization) {
		names = make([]string, len(groupIDs))

		for idx, groupID := range groupIDs {
			group, ok := org.Groups[groupID]
			if ok {
				names[idx] = group.Name
			}
		}
	})

	return
}

func (impl *storageImpl) SetGroupAdmin(groupID, personID uint64, adminFlag bool) error {
	return impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		if _, ok := newOrg.Persons[personID]; !ok {
			err = commerr.ErrNotFound

			return
		}

		group, ok := newOrg.Groups[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		var memberExists bool

		for _, memberID := range group.MemberPersonIDs {
			if memberID == personID {
				memberExists = true

				break
			}
		}

		if !memberExists {
			err = commerr.ErrNotFound

			return
		}

		for index, memberID := range group.AdminPersonIDs {
			if memberID == personID {
				if adminFlag {
					err = commerr.ErrAlreadyExists

					return
				}

				group.AdminPersonIDs = slices.Delete(group.AdminPersonIDs, index, index+1)

				newOrg.Groups[groupID] = group

				return
			}
		}

		if !adminFlag {
			err = commerr.ErrNotFound

			return
		}

		group.AdminPersonIDs = append(group.AdminPersonIDs, personID)

		newOrg.Groups[groupID] = group

		return
	})
}

func (impl *storageImpl) IsGroupAdmin(groupID, personID uint64) (adminFlag bool, err error) {
	impl.organization.Read(func(org *Organization) {
		group, ok := org.Groups[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		_, ok = org.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		var inGroup bool

		for _, id := range group.MemberPersonIDs {
			if id == personID {
				inGroup = true

				break
			}
		}

		if !inGroup {
			err = commerr.ErrNotFound

			return
		}

		for _, id := range group.AdminPersonIDs {
			if id == personID {
				adminFlag = true

				return
			}
		}
	})

	return
}

func (impl *storageImpl) NewWallet(name string, personID uint64) (id uint64, err error) {
	err = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		person, ok := newOrg.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		for _, wallet := range newOrg.SubWallets {
			if wallet.Name == name && wallet.PersonID == personID {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		id = snowflake.ID()

		newOrg.SubWallets[id] = model.Wallet{
			ID:       id,
			Name:     name,
			PersonID: personID,
		}

		person.SubWalletIDs = append(person.SubWalletIDs, id)

		newOrg.Persons[personID] = person

		return
	})

	return
}

func (impl *storageImpl) GetWallet(walletID uint64) (wallet model.Wallet, err error) {
	impl.organization.Read(func(org *Organization) {
		var ok bool

		wallet, ok = org.SubWallets[walletID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}
	})

	return
}

func (impl *storageImpl) GetPersonWalletIDs(personID uint64) (subWalletIDs []uint64, err error) {
	impl.organization.Read(func(org *Organization) {
		person, ok := org.Persons[personID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		subWalletIDs = person.SubWalletIDs
	})

	return
}

func (impl *storageImpl) SetPersonMerchant(personID uint64, costDir model.CostDir) error {
	return impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		newOrg.Merchants[personID] = costDir

		return
	})
}

func (impl *storageImpl) SetPersonGroupMerchant(personID, groupID uint64, costDir model.CostDir) error {
	return impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		if newOrg.GroupMerchants[groupID] == nil {
			newOrg.GroupMerchants[groupID] = make(map[uint64]model.CostDir)
		}

		newOrg.GroupMerchants[groupID][personID] = costDir

		return
	})
}

func (impl *storageImpl) GetMerchantPersons() (merchants []MerchantPersonInfo) {
	impl.organization.Read(func(org *Organization) {
		for u, dir := range org.Merchants {
			merchants = append(merchants, MerchantPersonInfo{
				PersonID: u,
				CostDir:  dir,
			})
		}
	})

	return
}

func (impl *storageImpl) IsMerchantPerson(personID uint64) (dir model.CostDir, ok bool) {
	impl.organization.Read(func(org *Organization) {
		dir, ok = org.Merchants[personID]
	})

	return
}

func (impl *storageImpl) GetGroupMerchantPersons(groupID uint64) (merchants []MerchantPersonInfo) {
	impl.organization.Read(func(org *Organization) {
		for u, dir := range org.GroupMerchants[groupID] {
			merchants = append(merchants, MerchantPersonInfo{
				PersonID: u,
				CostDir:  dir,
			})
		}
	})

	return
}

func (impl *storageImpl) IsGroupMerchantPerson(personID, groupID uint64) (dir model.CostDir, ok bool) {
	impl.organization.Read(func(org *Organization) {
		dir, ok = org.GroupMerchants[groupID][personID]
	})

	return
}

func (impl *storageImpl) NewLabel(name string) (id uint64, err error) {
	err = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		for _, label := range newOrg.Labels {
			if label.Name == name {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		id = snowflake.ID()

		newOrg.Labels[id] = model.Label{
			ID:   id,
			Name: name,
		}

		return
	})

	return
}

func (impl *storageImpl) GetLabels() (labels []model.Label, err error) {
	impl.organization.Read(func(org *Organization) {
		labels = make([]model.Label, 0, len(org.Labels))

		for _, label := range org.Labels {
			labels = append(labels, label)
		}
	})

	return
}

func (impl *storageImpl) GetLabelName(id uint64) (name string, err error) {
	impl.organization.Read(func(org *Organization) {
		label, ok := org.Labels[id]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		name = label.Name
	})

	return
}

func (impl *storageImpl) NewGroupLabel(groupID uint64, name string) (id uint64, err error) {
	err = impl.organization.Change(func(org *Organization) (newOrg *Organization, err error) {
		newOrg = org

		for _, label := range newOrg.GroupLabels[groupID] {
			if label.Name == name {
				err = commerr.ErrAlreadyExists

				return
			}
		}

		if len(newOrg.GroupLabels[groupID]) == 0 {
			newOrg.GroupLabels[groupID] = make(map[uint64]model.Label)
		}

		id = snowflake.ID()

		newOrg.GroupLabels[groupID][id] = model.Label{
			ID:   id,
			Name: name,
		}

		return
	})

	return
}

func (impl *storageImpl) GetGroupLabels(groupID uint64) (labels []model.Label, err error) {
	impl.organization.Read(func(org *Organization) {
		labels = make([]model.Label, 0, len(org.GroupLabels[groupID]))

		for _, label := range org.GroupLabels[groupID] {
			labels = append(labels, label)
		}
	})

	return
}

func (impl *storageImpl) GetGroupLabelName(labelID, groupID uint64) (name string, err error) {
	impl.organization.Read(func(org *Organization) {
		groups, ok := org.GroupLabels[groupID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		label, ok := groups[labelID]
		if !ok {
			err = commerr.ErrNotFound

			return
		}

		name = label.Name
	})

	return
}

func (impl *storageImpl) Record(groupID uint64, groupBill model.GroupBill) error {
	return impl.getGroupBills(groupID).AddBill(groupBill)
}

func (impl *storageImpl) GetBill(groupID uint64, billID string) (bill model.GroupBill, err error) {
	return impl.getGroupBills(groupID).GetBill(billID)
}

func (impl *storageImpl) DeleteRecord(groupID uint64, recordID string) error {
	return impl.getGroupBills(groupID).DeleteRecord(recordID)
}

func (impl *storageImpl) GetBills(groupID uint64) ([]model.GroupBill, error) {
	return impl.GetBillsEx(groupID, 0, 0, 0, 0, 0, 0)
}

func (impl *storageImpl) GetBillsEx(groupID uint64, startYear, startMonth, startDay, finishYear,
	finishMonth, finishDay int) ([]model.GroupBill, error) {
	var startDate, finishDate string

	if startYear > 0 {
		startDate = fmt.Sprintf("%04d%02d%02d", startYear, startMonth, startDay)
	}

	if finishYear > 0 {
		finishDate = fmt.Sprintf("%04d%02d%02d", finishYear, finishMonth, finishDay)
	}

	return impl.getGroupBills(groupID).GetBills(startDate, finishDate)
}

func (impl *storageImpl) GetBillsByID(groupID uint64, id string, count int, dirNew bool) ([]model.GroupBill, bool, error) {
	return impl.getGroupBills(groupID).ListBills(id, count, dirNew)
}

func (impl *storageImpl) GetDeletedBills(groupID uint64) (bills []model.DeletedGroupBill, err error) {
	return impl.getGroupBills(groupID).GetDeletedBills()
}

func (impl *storageImpl) GetDeletedBill(groupID uint64, billID string) (bill model.DeletedGroupBill, err error) {
	return impl.getGroupBills(groupID).GetDeletedBill(billID)
}

func (impl *storageImpl) CleanDeletedBill(groupID uint64, billID string) (err error) {
	return impl.getGroupBills(groupID).RemoveDeletedBillHistory(billID)
}

func (impl *storageImpl) RestoreDeletedBill(groupID uint64, billID string) (err error) {
	return impl.getGroupBills(groupID).RestoreDeletedBill(billID)
}

func (impl *storageImpl) key4GroupEnterCode(enterCode string) string {
	return "enter-code:" + enterCode
}

func (impl *storageImpl) AddGroupEnterCodes(enterCodes []string, personID, groupID uint64, duration time.Duration) (err error) {
	geInfo := &GroupEnterInfo{
		PersonID: personID,
		GroupID:  groupID,
	}

	d, _ := json.Marshal(geInfo)

	for _, code := range enterCodes {
		impl.tmpData.Set(impl.key4GroupEnterCode(code), d, duration)
	}

	err = impl.tmpData.SaveFile(impl.tmpDataFile)

	return
}

func (impl *storageImpl) ActiveGroupEnterCode(enterCode string) (personID, groupID uint64, ok bool, err error) {
	i, ok := impl.tmpData.Get(impl.key4GroupEnterCode(enterCode))
	if !ok {
		return
	}

	d, ok := i.([]byte)
	if !ok {
		return
	}

	ok = false

	var geInfo GroupEnterInfo

	err = json.Unmarshal(d, &geInfo)
	if err != nil {
		return
	}

	impl.tmpData.Delete(impl.key4GroupEnterCode(enterCode))

	err = impl.tmpData.SaveFile(impl.tmpDataFile)
	if err != nil {
		return
	}

	ok = true
	groupID = geInfo.GroupID
	personID = geInfo.PersonID

	return
}
