package jmshal

import (
	"encoding/binary"
	"errors"

	"github.com/BertoldVdb/jms578flash/image"
)

func (d *JMSHal) VersionGet() (uint32, error) {
	var cmdBuf [3]byte
	cmdBuf[0] = 0xe0
	cmdBuf[1] = 0xf4
	cmdBuf[2] = 0xe7

	var result [16]byte
	resultSlice := result[:]

	err := d.dev.Read(cmdBuf[:], &resultSlice)

	return binary.BigEndian.Uint32(result[12:]), err
}

const (
	regMapping8000 uint16 = 0x708c

	memBootWithoutRom uint16 = 0x4154
)

func (d *JMSHal) CodeWrite(buf []byte, tryVendorFirst bool) error {
	/* Vendor path corrupts 0x0000-0x0400 before overwriting it with valid data,
	 * as such you need to use raw writing with tryVendorFirst=false if there
	 * is already something running. */
	if tryVendorFirst {
		fwImage := image.Build(buf, nil, true)

		var cmdBuf [10]byte
		cmdBuf[0] = 0x3b
		cmdBuf[1] = 0x06

		binary.BigEndian.PutUint16(cmdBuf[7:], uint16(len(fwImage)))

		if err := d.dev.Write(cmdBuf[:], fwImage); err == nil {
			return d.reopen()
		}
	}

	if len(buf) > 0x4000 {
		return errors.New("code is too long for raw writing")
	}

	if err := d.XDATAWriteByte(regMapping8000, 6); err != nil {
		return err
	}

	if _, err := d.XDATAWrite(0x8000, buf); err != nil {
		return err
	}

	if _, err := d.XDATAWrite(memBootWithoutRom, []byte{'i', 's'}); err != nil {
		return err
	}

	return d.reopen()
}

func (d *JMSHal) codeRead(offset uint16, buf []byte) (int, error) {
	if len(buf) > 255 {
		buf = buf[:255]
	}

	workBuf := uint16(0x3600)

	ctx := CPUContext{}
	binary.BigEndian.PutUint16(ctx.R[6:], offset)
	binary.BigEndian.PutUint16(ctx.R[4:], workBuf)

	ctx, err := d.CodeCall(0x1f1b, ctx)
	if err != nil {
		return 0, err
	}

	return d.XDATARead(workBuf, buf)
}

func (d *JMSHal) CodeRead(offset uint16, buf []byte) (int, error) {
	return completeIO(offset, buf, d.codeRead)
}
