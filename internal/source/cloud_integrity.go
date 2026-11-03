package source

import "hash/crc32"

func validCRC(data []byte, want *int64) bool {
	return want != nil && uint64(*want) == uint64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
}
