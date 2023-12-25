package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/godruoyi/go-snowflake"
	"github.com/sgostarter/i/commerr"
)

func (s *Server) handleGroupNew(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	groupID, code, msg := s.handleGroupNewInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = GroupNewResponse{
			ID: idN2S(groupID),
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleGroupNewInner(c *gin.Context) (groupID uint64, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req GroupNewRequest

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

	groupID, err = s.storage.NewGroup(req.Name, uid)
	if err != nil {
		if errors.Is(err, commerr.ErrAlreadyExists) {
			code = CodeGroupNameExists
		} else {
			code = CodeInternalError
		}

		msg = err.Error()

		return
	}

	code = CodeSuccess

	return
}

func (s *Server) handleGroupEnterCodes(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	enterCodes, expireAt, code, msg := s.handleGroupEnterCodesInner(c)

	if code == CodeSuccess {
		respWrapper.Resp = GroupEnterCodesResponse{
			EnterCodes: enterCodes,
			ExpireAt:   expireAt.Unix(),
			ExpireAtS:  expireAt.Format("01/02 15:04"),
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleGroupEnterCodesInner(c *gin.Context) (enterCodes []string, expireAt time.Time, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req GroupEnterCodesRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	groupIDs, err := s.storage.GetPersonGroupsIDs(uid)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	if len(groupIDs) == 0 {
		code = CodeInternalError
		msg = "该用户不属于任何组"

		return
	}

	var groupID uint64

	if req.GroupID != "" {
		groupID, err = idS2N(req.GroupID)
		if err != nil {
			code = CodeInvalidArgs
			msg = err.Error()

			return
		}
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	var groupExists bool

	for _, gID := range groupIDs {
		if f, _ := s.storage.IsGroupAdmin(gID, uid); f {
			if groupID == 0 {
				groupID = gID
			}

			if groupID == gID {
				groupExists = true

				break
			}
		}
	}

	if !groupExists {
		code = CodeInvalidArgs
		msg = "无权限或者组不存在"

		return
	}

	if req.Count <= 0 {
		req.Count = 1
	} else if req.Count > 20 {
		req.Count = 20
	}

	code = CodeSuccess

	enterCodes = make([]string, 0)

	for idx := 0; idx < req.Count; idx++ {
		enterCodes = append(enterCodes, strconv.FormatUint(snowflake.ID(), 10))
	}

	expireAt = time.Now().Add(time.Hour)

	err = s.storage.AddGroupEnterCodes(enterCodes, uid, groupID, time.Hour)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	return
}

func (s *Server) handleGroupJoin(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleGroupJoinInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleGroupJoinInner(c *gin.Context) (code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	enterCode := c.Param("code")
	if enterCode == "" {
		code = CodeInvalidArgs
		msg = "no enter code"

		return
	}

	personID, groupID, ok, err := s.storage.ActiveGroupEnterCode(enterCode)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	if !ok {
		code = CodeInvalidArgs
		msg = "no valid enter code"

		return
	}

	adminFlag, err := s.storage.IsGroupAdmin(groupID, personID)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	if !adminFlag {
		code = CodeDisabled
		msg = "enter code has been disabled"

		return
	}

	err = s.storage.JoinGroup(groupID, uid)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	code = CodeSuccess

	return
}

// nolint: unused
func (s *Server) inGroupEx(uid, groupID uint64) (suggestGroupID uint64, ok bool) {
	groupIDs, _ := s.storage.GetPersonGroupsIDs(uid)
	if len(groupIDs) == 0 {
		return
	}

	suggestGroupID = groupID
	if suggestGroupID == 0 {
		suggestGroupID = groupIDs[0]
	}

	for _, d := range groupIDs {
		if d == suggestGroupID {
			ok = true

			return
		}
	}

	return
}

// nolint: unused
func (s *Server) inGroup(uid, groupID uint64) bool {
	if groupID == 0 {
		return false
	}

	_, ok := s.inGroupEx(uid, groupID)

	return ok
}
