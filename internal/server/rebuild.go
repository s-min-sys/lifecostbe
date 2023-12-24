package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/libcomponents/statistic/memdate"
	"github.com/sgostarter/libcomponents/statistic/memdate/ex"
	"github.com/sgostarter/libeasygo/stg/fs/rawfs"
	"github.com/sgostarter/libeasygo/stg/mwf"
	"github.com/spf13/cast"
)

/*
group-id:label-id:
group-id:group-id
*/

func billStatKey(groupID, labelID uint64) string {
	return fmt.Sprintf("%d-%d", groupID, labelID)
}

func readFileBills(path string) (bills []model.GroupBill, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		var line []byte

		line, err = reader.ReadBytes('\n')
		if err == io.EOF {
			err = nil

			break
		}

		if err != nil {
			return
		}

		var bill model.GroupBill

		err = json.Unmarshal(line, &bill)
		if err != nil {
			return
		}

		bills = append(bills, bill)
	}

	return
}

func bill2LifeCostData4Delete(bill model.GroupBill) ex.LifeCostData {
	curD := ex.LifeCostData{
		T: ex.ListCostDataDelete,
	}

	if bill.CostDir == model.CostDirIn {
		curD.EarnCount = 1
		curD.EarnAmount = bill.Amount
	} else if bill.CostDir == model.CostDirOut {
		curD.ConsumeCount = 1
		curD.ConsumeAmount = bill.Amount
	} else {
		curD.T = ex.ListCostDataNon
	}

	return curD
}

func bill2LifeCostData4Add(bill model.GroupBill) ex.LifeCostData {
	curD := ex.LifeCostData{
		T: ex.ListCostDataAdd,
	}

	if bill.CostDir == model.CostDirIn {
		curD.EarnCount = 1
		curD.EarnAmount = bill.Amount
	} else if bill.CostDir == model.CostDirOut {
		curD.ConsumeCount = 1
		curD.ConsumeAmount = bill.Amount
	} else {
		curD.T = ex.ListCostDataNon
	}

	return curD
}

func RebuildBills() (err error) {
	_ = os.RemoveAll(filepath.Join(dataRoot, statFileName))

	stat := memdate.NewMemDateStatistics[string, ex.LifeCostTotalData, ex.LifeCostData,
		ex.LifeCostDataTrans, mwf.Serial, mwf.Lock](&mwf.JSONSerial{}, &mwf.NoLock{}, time.Local,
		statFileName, rawfs.NewFSStorage(dataRoot))

	files, err := os.ReadDir(filepath.Join(dataRoot, "bills"))
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ps := strings.Split(file.Name(), "-")
		if len(ps) != 2 {
			continue
		}

		if len(ps[1]) != 8 {
			continue
		}

		groupID := cast.ToUint64(ps[0])

		bills, _ := readFileBills(filepath.Join(dataRoot, "bills", file.Name()))

		for _, bill := range bills {
			if bill.CostDir == model.CostDirInGroup {
				continue
			}

			curD := bill2LifeCostData4Add(bill)

			if len(bill.LabelIDs) > 0 {
				for _, labelID := range bill.LabelIDs {
					stat.SetDayData(billStatKey(groupID, labelID), time.Unix(bill.At, 0), curD)
				}
			} else {
				stat.SetDayData(billStatKey(groupID, 0), time.Unix(bill.At, 0), curD)
			}

			stat.SetDayData(billStatKey(groupID, groupID), time.Unix(bill.At, 0), curD)
		}
	}

	return
}
