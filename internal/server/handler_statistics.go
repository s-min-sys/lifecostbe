package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgostarter/libcomponents/statistic/memdate"
	"github.com/sgostarter/libcomponents/statistic/memdate/ex"
	"golang.org/x/exp/slices"
)

func statAllWeekDay(index int, weekDay *memdate.WeekD[ex.LifeCostTotalData]) StatWeekDay {
	return StatWeekDay{
		WeekDay:  index,
		MonthDay: weekDay.Day,
		Stat:     *weekDay.TotalT,
	}
}

func statAllWeekDays(weekDays map[int]*memdate.WeekD[ex.LifeCostTotalData]) []StatWeekDay {
	rWeekDays := make([]StatWeekDay, 0, 7)

	for index, weekDay := range weekDays {
		rWeekDays = append(rWeekDays, statAllWeekDay(index, weekDay))
	}

	slices.SortFunc(rWeekDays, func(a, b StatWeekDay) int {
		if a.MonthDay == b.MonthDay {
			return 0
		}

		if a.MonthDay < b.MonthDay {
			return -1
		}

		return 1
	})

	return rWeekDays
}

func statAllWeek(index int, week *memdate.WeekData[ex.LifeCostTotalData]) StatWeek {
	return StatWeek{
		Week: index,
		Stat: *week.TotalD,
		Days: statAllWeekDays(week.Day),
	}
}

func statAllWeeks(weeks map[int]*memdate.WeekData[ex.LifeCostTotalData]) []StatWeek {
	rWeeks := make([]StatWeek, 0, 6)

	for index, week := range weeks {
		rWeeks = append(rWeeks, statAllWeek(index, week))
	}

	slices.SortFunc(rWeeks, func(a, b StatWeek) int {
		if a.Week == b.Week {
			return 0
		}

		if a.Week < b.Week {
			return -1
		}

		return 1
	})

	return rWeeks
}

func statAllMonth(index int, month *memdate.MonthData[ex.LifeCostTotalData]) StatMonth {
	return StatMonth{
		Month: index,
		Stat:  *month.TotalD,
		Weeks: statAllWeeks(month.Week),
	}
}

func statAllMonths(months map[int]*memdate.MonthData[ex.LifeCostTotalData]) []StatMonth {
	rMonths := make([]StatMonth, 0, 3)

	for monthIndex, month := range months {
		rMonths = append(rMonths, statAllMonth(monthIndex, month))
	}

	slices.SortFunc(rMonths, func(a, b StatMonth) int {
		if a.Month == b.Month {
			return 0
		}

		if a.Month < b.Month {
			return -1
		}

		return 1
	})

	return rMonths
}

func statAllSeason(seasonIndex int, season *memdate.SeasonData[ex.LifeCostTotalData]) StatSeason {
	return StatSeason{
		Season: seasonIndex,
		Stat:   *season.TotalD,
		Months: statAllMonths(season.Month),
	}
}

func statAllSeasons(seasons map[int]*memdate.SeasonData[ex.LifeCostTotalData]) []StatSeason {
	rSeasons := make([]StatSeason, 0, 4)

	for seasonIndex, season := range seasons {
		rSeasons = append(rSeasons, statAllSeason(seasonIndex, season))
	}

	slices.SortFunc(rSeasons, func(a, b StatSeason) int {
		if a.Season == b.Season {
			return 0
		}

		if a.Season < b.Season {
			return -1
		}

		return 1
	})

	return rSeasons
}

func statAllYear(year int, d *memdate.YearData[ex.LifeCostTotalData]) StatYear {
	return StatYear{
		Year:    year,
		Stat:    *d.TotalD,
		Seasons: statAllSeasons(d.Season),
	}
}

func statAllYears(years map[int]*memdate.YearData[ex.LifeCostTotalData]) []StatYear {
	rYears := make([]StatYear, 0, 4)

	for index, year := range years {
		rYears = append(rYears, statAllYear(index, year))
	}

	slices.SortFunc(rYears, func(a, b StatYear) int {
		if a.Year == b.Year {
			return 0
		}

		if a.Year < b.Year {
			return -1
		}

		return 1
	})

	return rYears
}

func (s *Server) handleStatisticsAll(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	statYears, code, msg := s.handleStatisticsAllInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = StatAllResponse{
			Years: statYears,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleStatisticsAllInner(c *gin.Context) (years []StatYear, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	groupID, ok := s.inGroupEx(uid, 0)
	if !ok {
		code = CodeDisabled
		msg = "该用户不属于任何组"

		return
	}

	daM, err := s.stat.Export(billStatKey(groupID, groupID))
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	years = statAllYears(daM)

	return
}

func (s *Server) handleStatisticsNow(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	dayStatistics, weekStatistics, monthStatistics, seasonStatistics,
		yearStatistics, code, msg := s.handleStatisticsNowInner(c)
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

func (s *Server) handleStatisticsNowInner(c *gin.Context) (dayStatistics, weekStatistics,
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
