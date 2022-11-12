package jmshal

import (
	"bytes"
	"errors"

	"github.com/BertoldVdb/jms578flash/image"
	"github.com/BertoldVdb/jms578flash/jmsmods"
	"github.com/BertoldVdb/jms578flash/spiflash"

	_ "embed"
)

func (d *JMSHal) FlashWriteFirmware(fw []byte, verify bool) error {
	if !d.unsafe {
		return errors.New("flash write requires unsafeAllow=true")
	}

	if len(fw) < 0xc400 {
		return errors.New("firmware file too small")
	}

	flash, err := spiflash.New(d.SPI, d.SPIMaxTransactionSize())
	if err != nil {
		return err
	}

	if err := flash.EraseChip(); err != nil {
		return err
	}

	if _, err := flash.Write(0x0e00, fw[:0x200]); err != nil {
		return err
	}
	if _, err := flash.Write(0x0000, fw[0x200:0x400]); err != nil {
		return err
	}
	if _, err := flash.Write(0x1000, fw[0x400:0xc400]); err != nil {
		return err
	}
	if _, err = flash.Write(0xd000, fw[0xc400:]); err != nil {
		return err
	}

	if verify {
		rb, err := d.FlashReadFirmware()
		if err != nil {
			return err
		}

		if !bytes.Equal(rb, fw) {
			return errors.New("verify failed")
		}
	}

	return nil
}

func (d *JMSHal) FlashReadFirmware() ([]byte, error) {
	flash, err := spiflash.New(d.SPI, d.SPIMaxTransactionSize())
	if err != nil {
		return nil, err
	}

	fw := make([]byte, 0xc400+0x200)

	if _, err := flash.Read(0x0e00, fw[:0x200]); err != nil {
		return nil, err
	}
	if _, err := flash.Read(0x0000, fw[0x200:0x400]); err != nil {
		return nil, err
	}
	if _, err := flash.Read(0x1000, fw[0x400:0xc400]); err != nil {
		return nil, err
	}

	_, err = flash.Read(0xd000, fw[0xc400:])
	return fw, err
}

func (d *JMSHal) FlashEraseFirmware() error {
	if !d.unsafe {
		return errors.New("flash erase requires unsafeAllow=true")
	}

	flash, err := spiflash.New(d.SPI, d.SPIMaxTransactionSize())
	if err != nil {
		return err
	}

	return flash.ErasePage(0)
}

//go:embed asm/dumprom.bin
var dumprom []byte

func (d *JMSHal) DumpBootrom() ([]byte, error) {
	hdr := make([]byte, 0x50)
	for i := range hdr {
		hdr[i] = 0xff
	}

	err := d.FlashWriteFirmware(image.Build(append(hdr, dumprom...), nil, false), false)
	if err != nil {
		return nil, err
	}

	if err := d.ResetChip(); err != nil {
		return nil, err
	}

	rom := make([]byte, 0x4000)
	if _, err := d.XDATARead(0x8000, rom); err != nil {
		return nil, err
	}

	return rom, d.RebootToROM()
}

func (d *JMSHal) RebootToROM() error {
	if err := d.FlashEraseFirmware(); err != nil {
		return err
	}

	return d.ResetChip()
}

func (d *JMSHal) RebootToPatched(bootrom []byte) error {
	if d.PatchIsCurrent() {
		return nil
	}

	if d.unsafe {
		if version, err := d.VersionGet(); err != nil || version != 0 || d.hookVersion != "" {
			if err := d.RebootToROM(); err != nil {
				return err
			}
		}
	}

	patched, err := jmsmods.PatchBootromForHAL(bootrom)
	if err != nil {
		return err
	}

	/* If we cannot erase the firmware to force ROM mode,
	   we will try to load the patched bootrom via the firmware update mechanism.
	   This may fail and the device will crash. */
	if err := d.CodeWrite(patched, !d.unsafe, d.unsafe); err != nil {
		return err
	}

	if d.PatchIsCurrent() {
		return nil
	}

	return errors.New("patched bootrom did not start running")
}

/* This is the main function that does the whole flash procedure */
func (d *JMSHal) FlashPatchWriteAndBootFW(bootrom []byte, fw []byte, addHooks bool, mods []jmsmods.Mod, bootIt bool) error {
	fw, err := jmsmods.PatchCreate(fw, addHooks, mods)
	if err != nil {
		return err
	}

	if bootrom != nil {
		if err := d.RebootToPatched(bootrom); err != nil {
			return err
		}
	}

	currentFw, err := d.FlashReadFirmware()
	if err != nil {
		return err
	}

	if len(fw) > len(currentFw) {
		return errors.New("file is too large")
	}

	if bytes.Equal(fw, currentFw[:len(fw)]) {
		if version, err := d.VersionGet(); err != nil {
			return err
		} else if version == 0 {
			return d.ResetChip()
		}
		return nil
	}

	if err := d.FlashWriteFirmware(fw, true); err != nil {
		return err
	}

	if !bootIt {
		return nil
	}

	return d.ResetChip()
}
