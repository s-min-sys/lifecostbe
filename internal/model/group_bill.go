package model

import "time"

type CostDir int

const (
	CostDirInGroup CostDir = iota + 1
	CostDirIn
	CostDirOut
)

type GroupBill struct {
	ID                string   `json:"id"`
	FromSubWalletID   uint64   `json:"fromSubWalletID"`
	ToSubWalletID     uint64   `json:"toSubWalletID"`
	CostDir           CostDir  `json:"coastDir"`
	Amount            int      `json:"amount"`
	LabelIDs          []uint64 `json:"labelIDs"`
	Remark            string   `json:"remark"`
	LossAmount        int      `json:"lossAmount"`
	LossWalletID      uint64   `json:"lossWalletID"`
	At                int64    `json:"at"`
	OperationPersonID uint64   `json:"operationPersonID"`
}

func (gb *GroupBill) Valid() bool {
	if gb.FromSubWalletID == 0 || gb.ToSubWalletID == 0 || gb.Amount <= 0 ||
		gb.CostDir < CostDirInGroup || gb.CostDir > CostDirOut {
		return false
	}

	if gb.At <= 0 {
		gb.At = time.Now().Unix()
	}

	return true
}

type DeletedGroupBill struct {
	GroupBill `json:",inline"`
	DeletedAt time.Time `json:"deletedAt"`
}
