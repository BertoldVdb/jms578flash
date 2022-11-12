package jmsmods

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/BertoldVdb/jms578flash/image"
)

type Mod string

const (
	/* Make the firmware not write to the flash at all (so you can write protect it) */
	ModFlashNoWrite Mod = "FlashNoWrite"

	/* Make the firmware send the right commands for Adesto AT25DN512 flash */
	ModFlashSupportAT25DN512 Mod = "FlashSupportAT25DN512"

	/* Ignore any nvram present in the firmware file */
	ModClearNVRAM Mod = "ClearNVRAM"
)

func notSupported(h [20]byte, m Mod) error {
	return fmt.Errorf("mod %s not suppored on firmware with code checksum %s", m, hex.EncodeToString(h[:]))
}

// JMS578_STD_v00.04.01.04_Self Power + ODD.bin
var JMS578_414 = []byte{0x5e, 0x67, 0x7d, 0xaa, 0xc3, 0xdc, 0x3e, 0x31, 0xa0, 0x54, 0x81, 0x13, 0xf5, 0x60, 0x51, 0xde, 0x2e, 0x1d, 0x0b, 0x51}

func modsInstall(code []byte, nvram []byte, codeOffset uint16, mods []Mod) ([]byte, []byte, error) {
	h := sha1.Sum(code)

	for _, m := range mods {
		switch m {
		case ModFlashNoWrite:
			if bytes.Equal(h[:], JMS578_414) {
				code[0x5dfb-codeOffset] = 0x22
				continue
			}

		case ModFlashSupportAT25DN512:
			if bytes.Equal(h[:], JMS578_414) {
				copy(code[0x5f03-codeOffset:], []byte{0x2, 0x5f, 0xc9})
				continue
			}

		case ModClearNVRAM:
			for i := range nvram {
				nvram[i] = 0xff
			}
			continue

		default:
			return nil, nil, fmt.Errorf("unknown mod '%s'", m)
		}

		return nil, nil, notSupported(h, m)
	}

	return code, nvram, nil
}

func PatchCreate(fw []byte, addHooks bool, mods []Mod) ([]byte, error) {
	if !addHooks && len(mods) == 0 {
		return fw, nil
	}

	code, nvram, isRam, err := image.Extract(fw)
	if err != nil {
		return nil, err
	}

	if isRam {
		return nil, errors.New("cannot patch ram image")
	}

	code, nvram, err = modsInstall(code, nvram, 0x4000, mods)
	if err != nil {
		return nil, err
	}

	if addHooks {
		code, err = patchInstall(code, 0x4000, hooks)
		if err != nil {
			return nil, err
		}
	}

	return image.Build(code, nvram, isRam), nil
}
