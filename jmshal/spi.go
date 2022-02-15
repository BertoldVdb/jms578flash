package jmshal

import (
	"encoding/binary"
	"errors"
)

var ErrorSPIViolated = errors.New("SPI interface cannot handle transaction")

func (d *JMSHal) spiPIO(out []byte, in []byte) error {
	if len(out)+len(in) > 16 {
		return ErrorSPIViolated
	}

	/* Total transfer can be up to 16 bytes, first 'out' slice is sent, then
	 * 'in' slice is received */
	for _, m := range out {
		if err := d.XDATAWriteByte(0x7140, m); err != nil {
			return err
		}
	}

	/* Write readback scheme */
	for i := range in {
		if err := d.XDATAWriteByte(0x7141, byte(i)); err != nil {
			return err
		}
	}

	/* Start TXFR */
	if err := d.XDATAWriteByte(0x714c, 1); err != nil {
		return err
	}

	for {
		if value, err := d.XDATAReadByte(0x714c); err != nil {
			return err
		} else {
			if value == 0 {
				break
			}
		}
	}

	_, err := d.XDATARead(0x7150, in)
	return err
}

func (d *JMSHal) spiDMAInstalled() bool {
	return len(d.hooks) >= 3
}

func (d *JMSHal) spiDMATx(out []byte, in []byte) error {
	if !d.spiDMAInstalled() || len(out) > 512 || len(in) > 0 {
		return ErrorSPIViolated
	}

	workBuf := uint16(0x3700)

	if _, err := d.XDATAWrite(workBuf, out); err != nil {
		return err
	}

	ctx := CPUContext{}
	ctx.R[2] = 0x0
	ctx.R[3] = 0x20
	binary.LittleEndian.PutUint16(ctx.R[:], workBuf)
	binary.LittleEndian.PutUint16(ctx.R[4:], uint16(len(out)))

	_, err := d.hookCallIndex(2, ctx)
	return err
}

func (d *JMSHal) spiDMARx(out []byte, in []byte) error {
	//Note: the read is padded to a multiple of 4 bytes
	if !d.spiDMAInstalled() || len(out) > 14 || len(in) > 512 {
		return ErrorSPIViolated
	}

	workBuf := uint16(0x3700)

	if _, err := d.XDATAWrite(workBuf, out); err != nil {
		return err
	}

	ctx := CPUContext{}
	ctx.DPTR = workBuf
	binary.LittleEndian.PutUint16(ctx.R[:], workBuf)
	binary.LittleEndian.PutUint16(ctx.R[2:], uint16(len(in)))
	ctx.R[4] = byte(len(out))

	if _, err := d.hookCallIndex(1, ctx); err != nil {
		return err
	}

	_, err := d.XDATARead(workBuf, in)
	return err
}

func (d *JMSHal) SPI(out []byte, in []byte) error {
	/* Check which SPI implementation can do this */
	if err := d.spiDMATx(out, in); err == nil {
		return nil
	}
	if err := d.spiDMARx(out, in); err == nil {
		return nil
	}
	return d.spiPIO(out, in)
}

func (d *JMSHal) SPIMaxTransactionSize() int {
	if d.PatchIsPresent() {
		return 512
	}

	return 16
}
