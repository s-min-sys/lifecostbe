package server

func (s *Server) helperGetWalletName(walletID uint64) string {
	wallet, err := s.storage.GetWallet(walletID)
	if err != nil {
		return ""
	}

	return wallet.Name
}

func (s *Server) helperGetLabelName(labelID, personID uint64) string {
	name, err := s.storage.GetLabelName(labelID)
	if err == nil {
		return name
	}

	groupIDs, err := s.storage.GetPersonGroupsIDs(personID)
	if err != nil {
		return ""
	}

	for _, groupID := range groupIDs {
		name, err = s.storage.GetGroupLabelName(labelID, groupID)
		if err == nil {
			return name
		}
	}

	return ""
}

func (s *Server) helperGetLabelNames(labelIDs []uint64, personID uint64) (labelIDsS []string) {
	labelIDsS = make([]string, len(labelIDs))

	for idx := 0; idx < len(labelIDs); idx++ {
		labelIDsS[idx] = s.helperGetLabelName(labelIDs[idx], personID)
	}

	return
}

func (s *Server) helperPersonName(personID uint64) string {
	name, err := s.storage.GetPersonName(personID)
	if err != nil {
		return ""
	}

	return name
}

func (s *Server) helperGetWalletPersonName(walletID uint64) string {
	wallet, err := s.storage.GetWallet(walletID)
	if err != nil {
		return ""
	}

	name, _ := s.storage.GetPersonName(wallet.PersonID)

	return name
}
