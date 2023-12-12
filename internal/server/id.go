package server

import "strconv"

func idN2S(id uint64) string {
	return strconv.FormatUint(id, 16)
}

func idS2N(id string) (uint64, error) {
	return strconv.ParseUint(id, 16, 64)
}

func idN2Ss(ids []uint64) (rs []string) {
	rs = make([]string, len(ids))

	for idx, id := range ids {
		rs[idx] = idN2S(id)
	}

	return
}
