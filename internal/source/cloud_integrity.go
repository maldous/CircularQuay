package source

import "hash/crc32"

func validCRC(data []byte, want *int64) bool {
	if want == nil || *want < 0 {
		return false
	}
	return uint64(*want) == uint64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
}
