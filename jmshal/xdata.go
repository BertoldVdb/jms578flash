package jmshal

import (
	"encoding/binary"
)

func (d *JMSHal) xdataRead(offset uint16, buf []byte) (int, error) {
	var cmdBuf [12]byte
	cmdBuf[0] = 0xdf

	if len(buf) > 255 {
		buf = buf[:255]
	}

	cmdBuf[4] = byte(len(buf))

	binary.BigEndian.PutUint16(cmdBuf[6:], offset)

	/* This is the type of memory, the command can read flash as well,
	 * but we implement it ourselves to increase reliability */
	cmdBuf[11] = 0xfd

	err := d.dev.Read(cmdBuf[:], &buf)
	if err != nil {
		return 0, err
	}

	return len(buf), err
}

func (d *JMSHal) xdataWrite(offset uint16, buf []byte) (int, error) {
	var cmdBuf [12]byte
	cmdBuf[0] = 0xdf

	if len(buf) > 255 {
		buf = buf[:255]
	}

	cmdBuf[4] = byte(len(buf))

	binary.BigEndian.PutUint16(cmdBuf[6:], offset)

	cmdBuf[11] = 0xfe

	err := d.dev.Write(cmdBuf[:], buf)
	if err != nil {
		return 0, err
	}

	return len(buf), err
}

func completeIO(offset uint16, buf []byte, f func(offset uint16, buf []byte) (int, error)) (int, error) {
	if len(buf)+int(offset) > 0x10000 {
		buf = buf[:(0x10000 - int(offset))]
	}

	index := 0

	for len(buf) > 0 {
		n, err := f(offset, buf)
		index += n
		offset += uint16(n)

		if err != nil {
			return index, err
		}

		buf = buf[n:]
	}

	return index, nil
}

func (d *JMSHal) XDATARead(offset uint16, buf []byte) (int, error) {
	return completeIO(offset, buf, d.xdataRead)
}

func (d *JMSHal) XDATAWrite(offset uint16, buf []byte) (int, error) {
	return completeIO(offset, buf, d.xdataWrite)
}

func (d *JMSHal) XDATAReadByte(offset uint16) (byte, error) {
	var buf [1]byte

	_, err := d.XDATARead(offset, buf[:])
	return buf[0], err
}

func (d *JMSHal) XDATAWriteByte(offset uint16, value byte) error {
	var buf [1]byte
	buf[0] = value

	_, err := d.XDATAWrite(offset, buf[:])
	return err
}
