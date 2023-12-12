package model

type Person struct {
	ID           uint64   `json:"id"`
	Name         string   `json:"name"`
	Groups       []uint64 `json:"groups"`
	SubWalletIDs []uint64 `json:"subWalletIDs"`
}
