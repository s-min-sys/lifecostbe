package server

func (s *Server) getGroupID4Person(personID uint64, groupIDStr string) (selectedGroupID uint64, ok bool) {
	var groupID uint64
	var err error

	if groupIDStr != "" {
		groupID, err = idS2N(groupIDStr)
		if err != nil {
			return
		}
	}

	groups, err := s.storage.GetPersonGroupsIDs(personID)
	if err != nil {
		return
	}

	for _, gID := range groups {
		if groupID == 0 {
			groupID = gID
		}

		if groupID == gID {
			selectedGroupID = gID
			ok = true

			return
		}
	}

	return
}
