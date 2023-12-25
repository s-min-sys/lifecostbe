package server

import (
	"time"

	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/libcomponents/statistic/memdate/ex"
)

func (s *Server) statOnAddRecord(groupID uint64, labelIDs []uint64, groupBill model.GroupBill) {
	curD := bill2LifeCostData4Add(groupBill)
	if curD.T == ex.ListCostDataNon {
		return
	}

	if len(labelIDs) > 0 {
		for _, labelID := range labelIDs {
			s.stat.SetDayData(billStatKey(groupID, labelID), time.Unix(groupBill.At, 0), curD)
		}
	} else {
		s.stat.SetDayData(billStatKey(groupID, 0), time.Unix(groupBill.At, 0), curD)
	}

	s.stat.SetDayData(billStatKey(groupID, groupID), time.Unix(groupBill.At, 0), curD)
}

func (s *Server) statOnRemoveRecord(groupID uint64, groupBill model.GroupBill) {
	curD := bill2LifeCostData4Delete(groupBill)
	if curD.T == ex.ListCostDataNon {
		return
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

func (s *Server) getStats(groupID uint64, labelIDs []uint64) (
	dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics Statistics) {
	fnMergeStatistics := func(totalStat *Statistics, curStat Statistics) {
		totalStat.OutgoingCount += curStat.OutgoingCount
		totalStat.OutgoingAmount += curStat.OutgoingAmount
		totalStat.IncomingCount += curStat.IncomingCount
		totalStat.IncomingAmount += curStat.IncomingAmount
	}

	if len(labelIDs) == 0 {
		dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics = s.doStatistics(groupID, groupID)
	} else {
		for _, labelID := range labelIDs {
			d, w, m, s, y := s.doStatistics(groupID, labelID)
			fnMergeStatistics(&dayStatistics, d)
			fnMergeStatistics(&weekStatistics, w)
			fnMergeStatistics(&monthStatistics, m)
			fnMergeStatistics(&seasonStatistics, s)
			fnMergeStatistics(&yearStatistics, y)
		}
	}

	return
}
