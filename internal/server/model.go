package server

import (
	"time"

	"github.com/s-min-sys/lifecostbe/internal/model"
)

type ResponseWrapper struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Resp    interface{} `json:"resp,omitempty"`
}

func (wr *ResponseWrapper) Apply(code Code, msg string) {
	wr.Code = code
	wr.Message = CodeToMessage(code, msg)
}

type RegisterRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

func (req *RegisterRequest) Valid() bool {
	return req.UserName != "" && req.Password != ""
}

type RegisterResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type LoginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

func (req *LoginRequest) Valid() bool {
	return req.UserName != "" && req.Password != ""
}

type LoginResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type WalletWithInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MerchantWallets struct {
	PersonID   string            `json:"personID"`
	PersonName string            `json:"personName"`
	CostDir    model.CostDir     `json:"costDir"`
	Wallets    []*WalletWithInfo `json:"wallets"`
}

type PersonWallets struct {
	PersonID   string            `json:"personID"`
	PersonName string            `json:"personName"`
	Wallets    []*WalletWithInfo `json:"wallets"`
}

type IDName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GetBaseInfosResponse struct {
	MerchantWallets []MerchantWallets `json:"merchantWallets"`
	SelfWallets     MerchantWallets   `json:"selfWallets"`
	Labels          []IDName          `json:"labels"`
	Groups          []IDName          `json:"groups"`
}

type RecordRequest struct {
	FromSubWalletID string   `json:"fromSubWalletID"`
	ToSubWalletID   string   `json:"toSubWalletID"`
	Amount          int      `json:"amount"`
	LabelIDs        []string `json:"labelIDs"`
	Remark          string   `json:"remark"`
	LossAmount      int      `json:"lossAmount"`
	LossWalletID    string   `json:"lossWalletID"`
	At              int64    `json:"at"`

	DFromSubWalletID uint64   `json:"-"`
	DToSubWalletID   uint64   `json:"-"`
	DLabelIDs        []uint64 `json:"-"`
	DLossWalletID    uint64   `json:"-"`
}

func (req *RecordRequest) Valid() (ok bool) {
	var err error

	req.DFromSubWalletID, err = idS2N(req.FromSubWalletID)
	if err != nil {
		return
	}

	if req.DFromSubWalletID == 0 {
		return
	}

	req.DToSubWalletID, err = idS2N(req.ToSubWalletID)
	if err != nil {
		return
	}

	if req.DToSubWalletID == 0 {
		return
	}

	req.DLabelIDs = make([]uint64, 0, len(req.LabelIDs))

	for _, labelID := range req.LabelIDs {
		dLabelID, _ := idS2N(labelID)
		if dLabelID > 0 {
			req.DLabelIDs = append(req.DLabelIDs, dLabelID)
		}
	}

	if req.LossWalletID != "" {
		req.DLossWalletID, _ = idS2N(req.LossWalletID)
		if req.DLossWalletID == 0 {
			return
		}
	}

	if req.Amount <= 0 {
		return
	}

	if req.At == 0 {
		req.At = time.Now().Unix()
	}

	ok = true

	return
}

type WalletNewRequest struct {
	Name string `json:"name"`
}

func (req *WalletNewRequest) Valid() bool {
	return req.Name != ""
}

type WalletNewResponse struct {
	ID string `json:"id"`
}

type GroupNewRequest struct {
	Name string
}

func (req *GroupNewRequest) Valid() bool {
	return req.Name != ""
}

type GroupNewResponse struct {
	ID string `json:"id"`
}

type GetRecordsRequest struct {
	RecordID   string `json:"recordID"`
	PageCount  int    `json:"pageCount"`
	GroupID    string `json:"groupID"`
	NewForward bool   `json:"newForward"`
}

func (req *GetRecordsRequest) Valid() bool {
	return true
}

type Bill struct {
	ID                string        `json:"id"`
	FromSubWalletID   string        `json:"fromSubWalletID"`
	FromSubWalletName string        `json:"fromSubWalletName"`
	ToSubWalletID     string        `json:"toSubWalletID"`
	ToSubWalletName   string        `json:"toSubWalletName"`
	CostDir           model.CostDir `json:"costDir"`
	Amount            int           `json:"amount"`
	LabelIDs          []string      `json:"labelIDs"`
	LabelIDNames      []string      `json:"labelIDNames"`
	Remark            string        `json:"remark"`
	LossAmount        int           `json:"lossAmount"`
	LossWalletID      string        `json:"lossWalletID"`
	LossWalletName    string        `json:"lossWalletName"`
	At                int64         `json:"at"`
	AtS               string        `json:"atS"`

	FromPersonName string `json:"fromPersonName"`
	ToPersonName   string `json:"toPersonName"`
	OperationID    string `json:"operationID"`
	OperationName  string `json:"operationName"`
}

type GetRecordsResponse struct {
	Bills           []Bill     `json:"bills"`
	HasMore         bool       `json:"hasMore"`
	DayStatistics   Statistics `json:"dayStatistics"`
	WeekStatistics  Statistics `json:"weekStatistics"`
	MonthStatistics Statistics `json:"monthStatistics"`
}

type Statistics struct {
	IncomingCount   int `json:"incomingCount"`
	OutgoingCount   int `json:"outgoingCount"`
	GroupTransCount int `json:"groupTransCount"`

	IncomingAmount int `json:"incomingAmount"`
	OutgoingAmount int `json:"outgoingAmount"`
}

type StatisticsResponse struct {
	DayStatistics   Statistics `json:"dayStatistics"`
	WeekStatistics  Statistics `json:"weekStatistics"`
	MonthStatistics Statistics `json:"monthStatistics"`
}

type GroupEnterCodesRequest struct {
	GroupID string `json:"groupID"`
	Count   int    `json:"count"`
}

func (req *GroupEnterCodesRequest) Valid() bool {
	return true
}

type GroupEnterCodesResponse struct {
	EnterCodes []string `json:"enterCodes"`
	ExpireAt   int64    `json:"expireAt"`
	ExpireAtS  string   `json:"expireAtS"`
}

type WalletNewByDirRequest struct {
	GroupID       string        `json:"groupID"`
	NewWalletName string        `json:"newWalletName"`
	Dir           model.CostDir `json:"dir"`
}

func (req *WalletNewByDirRequest) Valid() bool {
	return req.NewWalletName != "" &&
		(req.Dir == model.CostDirIn || req.Dir == model.CostDirOut)
}

type BatchRecordRequest struct {
	Records []RecordRequest `json:"records"`
}

func (req *BatchRecordRequest) Valid() bool {
	for idx := range req.Records {
		if !req.Records[idx].Valid() {
			return false
		}
	}

	return true
}
