package jmstasks

import (
	"errors"

	"github.com/BertoldVdb/jms578flash/jmshal"
	"github.com/BertoldVdb/jms578flash/spiflash"
)

type JMSTasks struct {
	hal   *jmshal.JMSHal
	flash *spiflash.Flash
}

type JMSConfig struct {
	BootROM []byte
}

func New(hal *jmshal.JMSHal) (*JMSTasks, error) {
	flash, err := spiflash.New(hal.SPI, hal.SPIMaxTransactionSize())
	if err != nil {
		return nil, err
	}

	return &JMSTasks{
		hal:   hal,
		flash: flash,
	}, nil
}

func (t *JMSTasks) FirmwareWrite(fw []byte) error {
	if len(fw) < 0xc400 {
		return errors.New("firmware file too small")
	}

	if err := t.flash.EraseChip(); err != nil {
		return err
	}

	if _, err := t.flash.Write(0x0e00, fw[:0x200]); err != nil {
		return err
	}
	if _, err := t.flash.Write(0x0000, fw[0x200:0x400]); err != nil {
		return err
	}
	if _, err := t.flash.Write(0x1000, fw[0x400:0xc400]); err != nil {
		return err
	}
	_, err := t.flash.Write(0xd000, fw[0xc400:])

	return err
}

func (t *JMSTasks) ResetChip() error {
	return t.hal.ResetChip()
}
