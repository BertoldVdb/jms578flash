package image

import (
	"encoding/binary"

	"github.com/snksoft/crc"
)

var crcTable *crc.Table

func init() {
	params := crc.CRC32
	params.FinalXor = 0
	params.ReflectOut = false
	crcTable = crc.NewTable(params)
}

func crcCalculateBlock(data []byte) uint32 {
	if len(data)%4 > 0 {
		panic("block size needs to be a multiple of 4")
	}

	h := crc.NewHashWithTable(crcTable)

	var buf [4]byte
	for i := 0; i < len(data); i += 4 {
		buf[0] = data[i+3]
		buf[1] = data[i+2]
		buf[2] = data[i+1]
		buf[3] = data[i+0]
		h.Update(buf[:])
	}

	return h.CRC32()
}

func crcWriteCheck(slice []byte, value uint32, valid bool, doWrite bool) bool {
	if len(slice) < 4 {
		panic("slice length invalid")
	}

	orig := binary.BigEndian.Uint32(slice)
	if doWrite {
		binary.BigEndian.PutUint32(slice, value)
	}
	return orig == value && valid
}

func crcCalculateAndWriteCheck(block []byte, valid bool, doWrite bool) bool {
	crc := crcCalculateBlock(block[:len(block)-4])

	return crcWriteCheck(block[len(block)-4:], crc, valid, doWrite)
}
