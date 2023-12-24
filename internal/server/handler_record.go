package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libcomponents/statistic/memdate/ex"
	"golang.org/x/exp/slices"
)

func (s *Server) handleDeleteRecord(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleDeleteRecordInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleDeleteRecordInner(c *gin.Context) (code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	recordID := c.Param("id")
	if recordID == "" {
		code = CodeInternalError
		msg = "确实记录ID"

		return
	}

	groupIDs, err := s.storage.GetPersonGroupsIDs(uid)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	for _, groupID := range groupIDs {
		var groupBill model.GroupBill

		groupBill, err = s.storage.GetBill(groupID, recordID)
		if err != nil {
			s.logger.WithFields(l.ErrorField(err), l.UInt64Field("groupID", groupID),
				l.StringField("recordID", recordID)).Error("no bill")

			continue
		}

		err = s.storage.DeleteRecord(groupID, recordID)
		if err != nil {
			s.logger.WithFields(l.ErrorField(err)).Error("delete record failed")

			continue
		}

		curD := bill2LifeCostData4Delete(groupBill)
		if curD.T == ex.ListCostDataNon {
			continue
		}

		if len(groupBill.LabelIDs) > 0 {
			for _, labelID := range groupBill.LabelIDs {
				s.stat.SetDayData(billStatKey(groupID, labelID), time.Unix(groupBill.At, 0), curD)
			}
		} else {
			s.stat.SetDayData(billStatKey(groupID, 0), time.Unix(groupBill.At, 0), curD)
		}

		s.stat.SetDayData(billStatKey(groupID, groupID), time.Unix(groupBill.At, 0), curD)
	}

	return
}

func (s *Server) handleRecord(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleRecordInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) checkPersonCostDir(personID uint64) model.CostDir {
	dir, ok := s.storage.IsMerchantPerson(personID)
	if ok {
		return dir
	}

	groupIDs, err := s.storage.GetPersonGroupsIDs(personID)
	if err != nil {
		return model.CostDirInGroup
	}

	for _, groupID := range groupIDs {
		dir, ok = s.storage.IsGroupMerchantPerson(personID, groupID)
		if ok {
			return dir
		}
	}

	return model.CostDirInGroup
}

func (s *Server) handleRecordInner(c *gin.Context) (code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req RecordRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	return s.recordSingle(uid, req)
}

func (s *Server) recordSingle(uid uint64, req RecordRequest) (code Code, msg string) {
	fromWallet, err := s.storage.GetWallet(req.DFromSubWalletID)
	if err != nil {
		code = CodeInvalidArgs
		msg = err.Error()

		return
	}

	fromPersonGroupIDs, err := s.storage.GetPersonGroupsIDs(fromWallet.PersonID)
	if err != nil {
		code = CodeInvalidArgs
		msg = err.Error()

		return
	}

	toWallet, err := s.storage.GetWallet(req.DToSubWalletID)
	if err != nil {
		code = CodeInvalidArgs
		msg = err.Error()

		return
	}

	toPersonGroupIDs, err := s.storage.GetPersonGroupsIDs(toWallet.PersonID)
	if err != nil {
		code = CodeInvalidArgs
		msg = err.Error()

		return
	}

	var meInFrom, meInTo bool

	if fromWallet.PersonID == uid {
		meInFrom = true
	} else {
		if dir := s.checkPersonCostDir(toWallet.PersonID); dir == model.CostDirIn {
			code = CodeInvalidArgs
			msg = fmt.Sprintf("%s can't receive", toWallet.Name)

			return
		}
	}

	if toWallet.PersonID == uid {
		meInTo = true
	} else {
		if dir := s.checkPersonCostDir(fromWallet.PersonID); dir == model.CostDirIn {
			code = CodeInvalidArgs
			msg = fmt.Sprintf("%s can't send", toWallet.Name)

			return
		}
	}

	if !meInFrom && !meInTo {
		code = CodeInvalidArgs
		msg = "not your wallet"

		return
	}

	groupBill := model.GroupBill{
		FromSubWalletID:   req.DFromSubWalletID,
		ToSubWalletID:     req.DToSubWalletID,
		CostDir:           0,
		Amount:            req.Amount,
		LabelIDs:          req.DLabelIDs,
		Remark:            req.Remark,
		LossAmount:        req.LossAmount,
		LossWalletID:      req.DLossWalletID,
		At:                req.At,
		OperationPersonID: uid,
	}

	groupIDs, err := s.storage.GetPersonGroupsIDs(uid)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	for _, groupID := range groupIDs {
		inFromGroup := slices.Contains(fromPersonGroupIDs, groupID)
		inToGroup := slices.Contains(toPersonGroupIDs, groupID)

		if (inFromGroup && inToGroup) || (meInFrom && meInTo) {
			groupBill.CostDir = model.CostDirInGroup
		} else {
			if meInFrom {
				groupBill.CostDir = model.CostDirOut
			} else {
				groupBill.CostDir = model.CostDirIn
			}
		}

		if e := s.storage.Record(groupID, groupBill); e != nil {
			s.logger.WithFields(l.ErrorField(e)).Error("record failed")
		}

		curD := bill2LifeCostData4Add(groupBill)
		if curD.T == ex.ListCostDataNon {
			continue
		}

		if len(req.DLabelIDs) > 0 {
			for _, labelID := range req.DLabelIDs {
				s.stat.SetDayData(billStatKey(groupID, labelID), time.Unix(groupBill.At, 0), curD)
			}
		} else {
			s.stat.SetDayData(billStatKey(groupID, 0), time.Unix(groupBill.At, 0), curD)
		}

		s.stat.SetDayData(billStatKey(groupID, groupID), time.Unix(groupBill.At, 0), curD)
	}

	code = CodeSuccess

	return
}

func (s *Server) handleGetRecords(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	bills, hasMore, dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics, code, msg := s.handleGetRecordsInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = GetRecordsResponse{
			Bills:            bills,
			HasMore:          hasMore,
			DayStatistics:    dayStatistics,
			WeekStatistics:   weekStatistics,
			MonthStatistics:  monthStatistics,
			SeasonStatistics: seasonStatistics,
			YearStatistics:   yearStatistics,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleGetRecordsInner(c *gin.Context) (voBills []Bill, hasMore bool,
	dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics Statistics, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	withStatistics := c.Query("flag") == "1"

	var req GetRecordsRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	if !withStatistics {
		withStatistics = req.RequestStat
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	var groupID uint64

	if req.GroupID == "" || req.GroupID == "0" {
		groupIDs, _ := s.storage.GetPersonGroupsIDs(uid)
		if len(groupIDs) == 0 {
			code = CodeDisabled
			msg = "此此用户没加入任何组，无法记录"

			return
		}

		groupID = groupIDs[0]
	} else {
		groupID, err = idS2N(req.GroupID)
		if err != nil {
			code = CodeInvalidArgs
			msg = "非法的组ID"

			return
		}
	}

	bills, hasMore, err := s.storage.GetBillsByID(groupID, req.RecordID, req.PageCount, req.NewForward)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	for _, bill := range bills {
		voBills = append(voBills, Bill{
			ID:                bill.ID,
			FromSubWalletID:   idN2S(bill.FromSubWalletID),
			FromSubWalletName: s.helperGetWalletName(bill.FromSubWalletID),
			ToSubWalletID:     idN2S(bill.ToSubWalletID),
			ToSubWalletName:   s.helperGetWalletName(bill.ToSubWalletID),
			CostDir:           bill.CostDir,
			Amount:            bill.Amount,
			LabelIDs:          idN2Ss(bill.LabelIDs),
			LabelIDNames:      s.helperGetLabelNames(bill.LabelIDs, uid),
			Remark:            bill.Remark,
			LossAmount:        bill.LossAmount,
			LossWalletID:      idN2S(bill.LossWalletID),
			LossWalletName:    s.helperGetWalletName(bill.LossWalletID),
			At:                bill.At,
			AtS:               time.Unix(bill.At, 0).Format("01/02 15:04"),
			FromPersonName:    s.helperGetWalletPersonName(bill.FromSubWalletID),
			ToPersonName:      s.helperGetWalletPersonName(bill.ToSubWalletID),
			OperationID:       idN2S(bill.OperationPersonID),
			OperationName:     s.helperPersonName(bill.OperationPersonID),
		})
	}

	if withStatistics {
		fnMergeStatistics := func(totalStat *Statistics, curStat Statistics) {
			totalStat.OutgoingCount += curStat.OutgoingCount
			totalStat.OutgoingAmount += curStat.OutgoingAmount
			totalStat.IncomingCount += curStat.IncomingCount
			totalStat.IncomingAmount += curStat.IncomingAmount
		}

		if len(req.DStatLabelIDs) == 0 {
			dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics = s.doStatistics(groupID, groupID)
		} else {
			for _, labelID := range req.DStatLabelIDs {
				d, w, m, s, y := s.doStatistics(groupID, labelID)
				fnMergeStatistics(&dayStatistics, d)
				fnMergeStatistics(&weekStatistics, w)
				fnMergeStatistics(&monthStatistics, m)
				fnMergeStatistics(&seasonStatistics, s)
				fnMergeStatistics(&yearStatistics, y)
			}
		}
	}

	return
}

func (s *Server) handleBatchRecord(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleBatchRecordInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleBatchRecordInner(c *gin.Context) (code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req BatchRecordRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	var allMsg string

	for _, r := range req.Records {
		code, msg = s.recordSingle(uid, r)
		if code != CodeSuccess {
			s.logger.WithFields(l.ErrorField(err), l.StringField("msg", msg)).Error("record failed")

			allMsg += msg + "\n"
		}
	}

	code = CodeSuccess
	msg = allMsg

	return
}
