package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/s-min-sys/lifecostbe/internal/model"
	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/libeasygo/cuserror"
)

func (s *Server) handleWalletNew(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	walletID, code, msg := s.handleWalletNewInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = WalletNewResponse{
			ID: idN2S(walletID),
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleWalletNewInner(c *gin.Context) (walletID uint64, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req WalletNewRequest

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

	walletID, err = s.storage.NewWallet(req.Name, uid)
	if err != nil {
		if errors.Is(err, commerr.ErrAlreadyExists) {
			code = CodeWalletNameExists
			msg = "钱包已经存在"
		} else {
			code = CodeInternalError
			msg = err.Error()
		}

		return
	}

	code = CodeSuccess

	return
}

func (s *Server) handleWalletNewByDir(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleWalletNewByDirInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleWalletNewByDirInner(c *gin.Context) (code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	var req WalletNewByDirRequest

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

	dir := req.Dir

	groupID, ok := s.getGroupID4Person(uid, req.GroupID)
	if !ok {
		code = CodeInvalidArgs
		msg = "invalid group id"

		return
	}

	var merchantPersonID uint64

	merchants := s.storage.GetGroupMerchantPersons(groupID)
	for _, merchant := range merchants {
		if merchant.CostDir == dir {
			merchantPersonID = merchant.PersonID

			break
		}
	}

	if merchantPersonID == 0 {
		if dir == model.CostDirOut {
			merchantPersonID, err = s.newGroupMerchantPerson4Earn(groupID)
		} else if dir == model.CostDirIn {
			merchantPersonID, err = s.newGroupMerchantPerson4Consume(groupID)
		} else {
			err = cuserror.NewWithErrorMsg("invalid param")
		}
	}

	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	//

	_, err = s.storage.NewWallet(req.NewWalletName, merchantPersonID)
	if err != nil {
		if errors.Is(err, commerr.ErrAlreadyExists) {
			code = CodeWalletNameExists
		} else {
			code = CodeInternalError
			msg = err.Error()
		}

		return
	}

	code = CodeSuccess

	return
}

func (s *Server) newGroupMerchantPerson4Earn(groupID uint64) (personID uint64, err error) {
	earnID, personID, err := s.storage.NewPerson("进账")
	if err != nil {
		return
	}

	err = s.storage.SetPersonGroupMerchant(earnID, groupID, model.CostDirOut)

	return
}

func (s *Server) newGroupMerchantPerson4Consume(groupID uint64) (personID uint64, err error) {
	earnID, personID, err := s.storage.NewPerson("消费")
	if err != nil {
		return
	}

	err = s.storage.SetPersonGroupMerchant(earnID, groupID, model.CostDirIn)

	return
}
