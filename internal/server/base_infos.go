package server

import (
	"github.com/s-min-sys/lifecostbe/internal/storage"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/s-min-sys/lifecostbe/internal/model"
)

func (s *Server) handleGetBaseInfos(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	labels, groups, merchantWallets, selfWallets, code, msg := s.handleGetBaseINfosInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = GetBaseInfosResponse{
			MerchantWallets: merchantWallets,
			SelfWallets:     selfWallets,
			Labels:          labels,
			Groups:          groups,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) buildMerchantWallets(info storage.MerchantPersonInfo) (merchantWallets MerchantWallets, err error) {
	name, err := s.storage.GetPersonName(info.PersonID)
	if err != nil {
		return
	}

	merchantWallets = MerchantWallets{
		PersonID:   idN2S(info.PersonID),
		PersonName: name,
		CostDir:    info.CostDir,
		Wallets:    make([]*WalletWithInfo, 0, 2),
	}

	wallet, err := s.storage.GetWallet(info.PersonID)
	if err == nil {
		merchantWallets.Wallets = append(merchantWallets.Wallets, &WalletWithInfo{
			ID:   idN2S(wallet.ID),
			Name: wallet.Name,
		})
	}

	walletIDs, _ := s.storage.GetPersonWalletIDs(info.PersonID)
	for _, walletID := range walletIDs {
		if walletID == info.PersonID {
			continue
		}

		wallet, err = s.storage.GetWallet(walletID)
		if err != nil {
			continue
		}

		merchantWallets.Wallets = append(merchantWallets.Wallets, &WalletWithInfo{
			ID:   idN2S(wallet.ID),
			Name: wallet.Name,
		})
	}

	return
}

func (s *Server) getMerchantWallets(groupIDs []uint64) (wallets []MerchantWallets) {
	for _, info := range s.storage.GetMerchantPersons() {
		merchantWallet, err := s.buildMerchantWallets(info)
		if err != nil {
			continue
		}

		wallets = append(wallets, merchantWallet)
	}

	for _, groupID := range groupIDs {
		for _, info := range s.storage.GetGroupMerchantPersons(groupID) {
			merchantWallet, err := s.buildMerchantWallets(info)
			if err != nil {
				continue
			}

			wallets = append(wallets, merchantWallet)
		}
	}

	return
}

func (s *Server) handleGetBaseINfosInner(c *gin.Context) (labels, groups []IDName,
	merchantWallets []MerchantWallets, selfWallets MerchantWallets, code Code, msg string) {
	_, uid, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	selfWallets, err := s.buildMerchantWallets(storage.MerchantPersonInfo{
		PersonID: uid,
		CostDir:  model.CostDirInGroup,
	})
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	groupIDs, _ := s.storage.GetPersonGroupsIDs(uid)

	merchantWallets = s.getMerchantWallets(groupIDs)

	//
	//
	//

	dbLabels, _ := s.storage.GetLabels()

	for _, id := range groupIDs {
		groupLabels, e := s.storage.GetGroupLabels(id)
		if e != nil {
			continue
		}

		dbLabels = append(dbLabels, groupLabels...)
	}

	labels = make([]IDName, 0, len(dbLabels))

	for _, label := range dbLabels {
		labels = append(labels, IDName{
			ID:   idN2S(label.ID),
			Name: label.Name,
		})
	}

	//
	//
	//

	groupNames, err := s.storage.GetGroupNames(groupIDs)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	for idx, groupID := range groupIDs {
		groups = append(groups, IDName{
			ID:   idN2S(groupID),
			Name: groupNames[idx],
		})
	}

	code = CodeSuccess

	return
}
