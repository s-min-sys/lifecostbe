package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/pathutils"
	"golang.org/x/exp/slices"
)

func (impl *billFileImpl) deletedBillsFilePath() string {
	return filepath.Join(impl.dir, fmt.Sprintf("%d-%s", impl.groupID, deletedFileName))
}

func (impl *billFileImpl) saveDeletedBillsToFile(groupID uint64) (err error) {
	impl.mustDeletedBillsHistoryLoaded()

	d, err := json.Marshal(impl.deletedBills[groupID])
	if err != nil {
		return
	}

	filePath := impl.deletedBillsFilePath()

	_ = pathutils.MustDirOfFileExists(filePath)

	err = os.WriteFile(filePath, d, 0600)

	return
}

func (impl *billFileImpl) mustDeletedBillsHistoryLoaded() {
	if !impl.isDeletedBillsHistoryLoaded() {
		_ = impl.loadDeletedBillsHistory()
	}
}

func (impl *billFileImpl) isDeletedBillsHistoryLoaded() bool {
	_, ok := impl.deletedBills[impl.groupID]

	return ok
}

func (impl *billFileImpl) loadDeletedBillsHistory() (err error) {
	d, err := os.ReadFile(impl.deletedBillsFilePath())
	if err != nil {
		impl.deletedBills[impl.groupID] = make(map[string]model.DeletedGroupBill)

		return
	}

	var deletedBills map[string]model.DeletedGroupBill

	err = json.Unmarshal(d, &deletedBills)
	if err != nil {
		return
	}

	if len(deletedBills) == 0 {
		deletedBills = make(map[string]model.DeletedGroupBill)
	}

	impl.deletedBills[impl.groupID] = deletedBills

	return
}

func (impl *billFileImpl) deletedBillsM2S(billsM map[string]model.DeletedGroupBill) (bills []model.DeletedGroupBill) {
	for _, bill := range billsM {
		bills = append(bills, bill)
	}

	slices.SortFunc(bills, func(a, b model.DeletedGroupBill) int {
		if a.DeletedAt == b.DeletedAt {
			return 0
		}

		if a.DeletedAt.Before(b.DeletedAt) {
			return 1
		}

		return -1
	})

	return
}

//
//
//

func (impl *billFileImpl) getDeleteBill(billID string) (bill model.DeletedGroupBill, ok bool) {
	impl.mustDeletedBillsHistoryLoaded()

	impl.deletedBillsLock.Lock()
	defer impl.deletedBillsLock.Unlock()

	bills, ok := impl.deletedBills[impl.groupID]
	if !ok {
		return
	}

	bill, ok = bills[billID]

	return
}

func (impl *billFileImpl) getDeletedBills() (bills []model.DeletedGroupBill, err error) {
	impl.mustDeletedBillsHistoryLoaded()

	impl.deletedBillsLock.Lock()
	defer impl.deletedBillsLock.Unlock()

	billsM, ok := impl.deletedBills[impl.groupID]
	if !ok {
		return
	}

	bills = impl.deletedBillsM2S(billsM)

	return
}

func (impl *billFileImpl) addDeletedBill(bill model.DeletedGroupBill) (err error) {
	impl.mustDeletedBillsHistoryLoaded()

	impl.deletedBillsLock.Lock()
	defer impl.deletedBillsLock.Unlock()

	impl.deletedBills[impl.groupID][bill.ID] = bill

	err = impl.saveDeletedBillsToFile(impl.groupID)

	return
}

func (impl *billFileImpl) deleteDeletedBill(billID string) (err error) {
	impl.mustDeletedBillsHistoryLoaded()

	impl.deletedBillsLock.Lock()
	defer impl.deletedBillsLock.Unlock()

	if bills, ok := impl.deletedBills[impl.groupID]; ok {
		delete(bills, billID)

		err = impl.saveDeletedBillsToFile(impl.groupID)
	}

	return
}

func (impl *billFileImpl) restoreDeletedBill(billID string) (err error) {
	bill, ok := impl.getDeleteBill(billID)
	if !ok {
		err = commerr.ErrNotFound

		return
	}

	impl.logger.WithFields(l.StringField("billID", billID)).Info("restore bill before")

	err = impl.AddBill(bill.GroupBill)

	impl.logger.WithFields(l.StringField("billID", billID)).Info("restore bill after")

	if err != nil {
		impl.logger.WithFields(l.StringField("billID", billID)).Error("restore bill failed")

		return
	}

	err = impl.deleteDeletedBill(billID)
	if err != nil {
		impl.logger.WithFields(l.StringField("billID", billID)).Error("delete bill history failed")

		return
	}

	return
}
