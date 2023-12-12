package model

type Wallet struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	PersonID uint64 `json:"personID"`
}
