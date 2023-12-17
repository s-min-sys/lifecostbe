package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/now"
	"github.com/s-min-sys/lifecostbe/internal/model"
	"net/http"
	"time"
)

func (s *Server) handleStatistics(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	dayStatistics, weekStatistics, monthStatistics, code, msg := s.handleStatisticsInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = StatisticsResponse{
			DayStatistics:   dayStatistics,
			WeekStatistics:  weekStatistics,
			MonthStatistics: monthStatistics,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleStatisticsInner(c *gin.Context) (dayStatistics, weekStatistics, monthStatistics Statistics,
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

	dayStatistics, weekStatistics, monthStatistics, err := s.doStatistics(groupID)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	return
}

func (s *Server) doStatisticsOnDates(groupID uint64, dateStart, dateEnd time.Time) (statistics Statistics, err error) {
	bills, err := s.storage.GetBillsEx(groupID, dateStart.Year(), int(dateStart.Month()), dateStart.Day(),
		dateEnd.Year(), int(dateEnd.Month()), dateEnd.Day())
	if err != nil {
		return
	}

	statistics = Statistics{}

	for _, bill := range bills {
		switch bill.CostDir {
		case model.CostDirInGroup:
			statistics.GroupTransCount++
		case model.CostDirIn:
			statistics.IncomingCount++
			statistics.IncomingAmount += bill.Amount
		case model.CostDirOut:
			statistics.OutgoingCount++
			statistics.OutgoingAmount += bill.Amount
		}
	}

	return
}

func (s *Server) doStatistics(groupID uint64) (dayStatistics, weekStatistics, monthStatistics Statistics, err error) {
	timeNow := time.Now()

	dayStatistics, err = s.doStatisticsOnDates(groupID, timeNow, timeNow)
	if err != nil {
		return
	}

	n := now.With(timeNow)

	n.WeekStartDay = time.Monday

	weekStatistics, err = s.doStatisticsOnDates(groupID,
		n.BeginningOfWeek(), n.EndOfWeek())
	if err != nil {
		return
	}

	monthStatistics, err = s.doStatisticsOnDates(groupID,
		n.BeginningOfMonth(), n.EndOfMonth())

	return
}
