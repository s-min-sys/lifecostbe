package model

type Group struct {
	ID              uint64   `json:"id"`
	Name            string   `json:"name"`
	MemberPersonIDs []uint64 `json:"memberPersonIDs"`
	AdminPersonIDs  []uint64 `json:"adminPersonIDs"`
}
