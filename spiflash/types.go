package spiflash

type flashDevice struct {
	deviceID uint32
	name     string

	opcodeChipErase  uint8
	opcodeBlockErase uint8
	opcodePageErase  uint8

	blockSize uint32
	pageSize  uint32
	chipSize  uint32
}

var devices = []flashDevice{
	{deviceID: 0x1f65, name: "Adesto AT25DN512", opcodeChipErase: 0x60, opcodeBlockErase: 0x20, blockSize: 4096, opcodePageErase: 0x81, pageSize: 256, chipSize: 64 * 1024},
	{deviceID: 0xef3012, name: "Winbond W25X20", opcodeChipErase: 0xC7, opcodeBlockErase: 0x20, blockSize: 4096, opcodePageErase: 0xD8, pageSize: 256, chipSize: 256 * 1024},
}

func rightAlign(in uint32) (uint32, uint32) {
	mask := uint32(0)

	for (in >> 24) == 0 {
		in <<= 8
		mask <<= 8
		mask |= 0xFF
	}
	return in, ^mask
}

func deviceLookup(id uint32) (flashDevice, bool) {
	for _, m := range devices {
		compare, mask := rightAlign(m.deviceID)

		if id&mask == compare {
			return m, true
		}
	}
	return devices[0], false
}
