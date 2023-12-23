package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgostarter/libcomponents/statistic/memdate/ex"
)

func (s *Server) handleStatistics(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics, code, msg := s.handleStatisticsInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = StatisticsResponse{
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

func (s *Server) handleStatisticsInner(c *gin.Context) (dayStatistics, weekStatistics,
	monthStatistics, seasonStatistics, yearStatistics Statistics,
	code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	groupIDs, _ := s.storage.GetPersonGroupsIDs(uid)
	if len(groupIDs) == 0 {
		code = CodeDisabled
		msg = "此此用户没加入任何组，无法记录"

		return
	}

	groupID := groupIDs[0]

	dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics = s.doStatistics(groupID, groupID)

	return
}

func (s *Server) doStatistics(groupID uint64, labelID uint64) (dayStatistics, weekStatistics, monthStatistics, seasonStatistics, yearStatistics Statistics) {
	timeNow := time.Now()

	fnTotalD2Statistic := func(totalD ex.LifeCostTotalData) Statistics {
		return Statistics{
			IncomingCount:  totalD.EarnCount,
			OutgoingCount:  totalD.ConsumeCount,
			IncomingAmount: totalD.EarnAmount,
			OutgoingAmount: totalD.ConsumeAmount,
		}
	}

	totalD, exists := s.stat.GetYearOn(billStatKey(groupID, labelID), timeNow)
	if exists {
		yearStatistics = fnTotalD2Statistic(totalD)
	}

	totalD, exists = s.stat.GetSeasonOn(billStatKey(groupID, labelID), timeNow)
	if exists {
		seasonStatistics = fnTotalD2Statistic(totalD)
	}

	totalD, exists = s.stat.GetMonthOn(billStatKey(groupID, labelID), timeNow)
	if exists {
		monthStatistics = fnTotalD2Statistic(totalD)
	}

	totalD, exists = s.stat.GetWeekOn(billStatKey(groupID, labelID), timeNow)
	if exists {
		weekStatistics = fnTotalD2Statistic(totalD)
	}

	totalD, exists = s.stat.GetDayOn(billStatKey(groupID, labelID), timeNow)
	if exists {
		dayStatistics = fnTotalD2Statistic(totalD)
	}

	return
}
