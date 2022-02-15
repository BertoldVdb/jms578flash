package jmshal

import (
	"bytes"
	"errors"
	"log"

	"github.com/BertoldVdb/jms578flash/image"
	"github.com/BertoldVdb/jms578flash/spiflash"

	_ "embed"
)

func (d *JMSHal) FlashWriteFirmware(fw []byte, verify bool) error {
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

func (t *JMSHal) FlashEraseFirmware() error {
	flash, err := spiflash.New(t.SPI, t.SPIMaxTransactionSize())
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

	return rom, d.GoROM()
}

func (d *JMSHal) GoROM() error {
	if err := d.FlashEraseFirmware(); err != nil {
		return err
	}

	return d.ResetChip()
}

func (d *JMSHal) GoPatched(bootrom []byte) error {
	if d.PatchIsPresent() {
		return nil
	}

	if version, err := d.VersionGet(); err != nil || version != 0 || d.hookVersion != "" {
		if err := d.GoROM(); err != nil {
			return err
		}
	}

	patched, err := patchBootromForHAL(bootrom)
	if err != nil {
		log.Fatalln(err)
	}

	return d.CodeWrite(patched, false)
}

/* This is the main function that does the whole flash procedure */
func (d *JMSHal) FlashInstallPatchAndBootFW(bootrom []byte, fw []byte) error {
	code, nvram, isRam, err := image.Extract(fw)
	if err != nil {
		return err
	}

	if isRam {
		return errors.New("cannot write ram image")
	}

	code, err = patchInstall(code, 0x4000, hooks)
	if err != nil {
		return err
	}

	if false {
		//TODO: Make this mod sha256 dependent!!!
		//Do not do any writing since this flash does not properly erase
		code[0x5dfb-0x4000] = 0x22
	} else {
		//TODO: Alternative option to put the correct flash commands (0x5fc9)
		copy(code[0x5f03-0x4000:], []byte{0x2, 0x5f, 0xc9})
	}

	fw = image.Build(code, nvram, isRam)

	if bootrom != nil {
		if err := d.GoPatched(bootrom); err != nil {
			return err
		}
	}

	currentFw, err := d.FlashReadFirmware()
	if err != nil {
		return err
	}

	if bytes.Equal(fw, currentFw) {
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

	return d.ResetChip()
}
