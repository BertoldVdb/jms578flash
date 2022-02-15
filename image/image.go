package image

import (
	"bytes"
	"encoding/binary"
	"errors"
)

func makeHeader(fw []byte, isRam bool) {
	/* Write header in block 0 */
	fw[0] = 1
	fw[1] = 0
	binary.BigEndian.PutUint32(fw[2:], 0x152d0579)

	if isRam {
		binary.BigEndian.PutUint32(fw[6:], 0x04040606)
	} else {
		binary.BigEndian.PutUint32(fw[6:], 0x03030505)
	}

	copy(fw[10:], []byte("JMicron JMS579"))
}

func checksumInternal(input []byte, isRam bool, doWrite bool) bool {
	wasValid := true

	/* Fix metadata block 1 */
	if isRam {
		wasValid = crcCalculateAndWriteCheck(input[0:0x200-4], wasValid, doWrite)
	}
	wasValid = crcCalculateAndWriteCheck(input[0:0x200], wasValid, doWrite)

	/* Fix firmware CRC */
	wasValid = crcCalculateAndWriteCheck(input[0x400:0xC400-4], wasValid, doWrite)

	if isRam {
		/* Write metadata CRC */
		wasValid = crcCalculateAndWriteCheck(input[:0x400], wasValid, doWrite)

		/* Write full file CRC */
		wasValid = crcCalculateAndWriteCheck(input[:0xC400], wasValid, doWrite)
		return wasValid
	}

	/* Zero out metadata block 2 to compute full CRC */
	work := make([]byte, len(input))
	copy(work, input)
	for i := 0x200; i < 0x400; i++ {
		work[i] = 0
	}

	fullCRC := crcCalculateBlock(work[:0xC400-4])

	/* Compensate the zeroing */
	fullCRC ^= uint32(0x7da476e9)

	/* Write full CRC at end of file */
	wasValid = crcWriteCheck(input[0xC400-4:], fullCRC, wasValid, doWrite)

	/* Fill in metadata block 2 */
	wasValid = crcWriteCheck(input[0x208:], fullCRC, wasValid, doWrite)
	wasValid = crcCalculateAndWriteCheck(input[0:0x400], wasValid, doWrite)

	return wasValid
}

func ChecksumUpdate(input []byte, isRam bool) bool {
	return checksumInternal(input, isRam, true)
}

var (
	ErrorInvalidLength = errors.New("image length not valid")
	ErrorInvalidHeader = errors.New("header is not valid")
	ErrorInvalidCRC    = errors.New("CRC is not valid")
)

func Validate(image []byte, isRam bool) error {
	if isRam && len(image) != 0xC400 {
		return ErrorInvalidLength
	}
	if !isRam && len(image) != 0xC400 && len(image) != 0xC600 {
		return ErrorInvalidLength
	}

	var hdr [0x18]byte
	makeHeader(hdr[:], isRam)

	if !bytes.Equal(hdr[:], image[:len(hdr)]) {
		return ErrorInvalidHeader
	}

	if !checksumInternal(image, isRam, false) {
		return ErrorInvalidCRC
	}

	return nil
}

func Build(code []byte, nvram []byte, isRam bool) []byte {
	length := 0xc400
	if !isRam {
		length += 0x200
	}

	fw := make([]byte, length)
	for i := 0x200; i < len(fw); i++ {
		fw[i] = 0xff
	}

	makeHeader(fw, isRam)

	/* Write the version (windows tool is picky on what it accepts) */
	fw[0x18] = 1
	copy(fw[0x19:], []byte("0103"))

	/* Write magic in block 1 */
	binary.BigEndian.PutUint32(fw[0x200:], 0x5ac369e1)

	if copy(fw[0x400:0xc400-8], code) < len(code) {
		panic("code buffer too long")
	}

	ChecksumUpdate(fw, isRam)

	if !isRam && nvram != nil {
		if copy(fw[0xc400:], nvram) < len(nvram) {
			panic("nvram buffer too long")
		}
	}

	return fw
}

func Extract(image []byte) ([]byte, []byte, bool, error) {
	if len(image) < 10 {
		return nil, nil, false, ErrorInvalidLength
	}

	isRam := binary.BigEndian.Uint32(image[6:]) == 0x04040606
	if err := Validate(image, isRam); err != nil {
		return nil, nil, isRam, err
	}

	return image[0x400 : 0xC400-8], image[0xC400:], isRam, nil
}
