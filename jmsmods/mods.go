package jmsmods

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/BertoldVdb/jms578flash/image"

	_ "embed"
)

type Mod string

const (
	/* Make the firmware not write to the flash at all (so you can write protect it) */
	ModFlashNoWrite Mod = "FlashNoWrite"

	/* Make the firmware send the right commands for Adesto AT25DN512 flash */
	ModFlashSupportAT25DN512 Mod = "FlashSupportAT25DN512"

	/* Ignore any nvram present in the firmware file */
	ModClearNVRAM Mod = "ClearNVRAM"

	/* Try to remove all debug commands to secure the device */
	ModNoDebug Mod = "NoDebug"

	/* Add hook to firmware to allow this library to work without rebooting to our own stub */
	ModAddHooks Mod = "AddHooks"
)

func notSupported(h [20]byte, m Mod) error {
	return fmt.Errorf("mod %s not suppored on firmware with code checksum %s", m, hex.EncodeToString(h[:]))
}

// JMS578_STD_v00.04.01.04_Self Power + ODD.bin
var JMS578_414 = []byte{0x5e, 0x67, 0x7d, 0xaa, 0xc3, 0xdc, 0x3e, 0x31, 0xa0, 0x54, 0x81, 0x13, 0xf5, 0x60, 0x51, 0xde, 0x2e, 0x1d, 0x0b, 0x51}

//go:embed asm/disable.bin
var disabledHandler []byte

func modsInstall(code []byte, nvram []byte, codeOffset uint16, mods []Mod) ([]byte, []byte, error) {
	h := sha1.Sum(code)

	hookImpossible := false

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

		case ModNoDebug:
			if hookImpossible {
				return nil, nil, errors.New("mod conflict (ModNoDebug/ModAddHooks)")
			}
			hookImpossible = true

			/* Write empty response stub */
			patchLoadAddr := patchFindLoadAddress(code)
			copy(code[patchLoadAddr:], disabledHandler)

			/* Try to find the table with commands */
			table, err := patchFindJumpTable(code)
			if err != nil {
				return nil, nil, err
			}

			/* Replace all known dangerous commands with empty handler */
			replaceCommand := func(types []uint8) {
				for _, t := range types {
					for _, m := range table {
						if m.Type == t {
							binary.BigEndian.PutUint16(code[m.AddrEntry:], patchLoadAddr+codeOffset)
							break
						}
					}
				}
			}

			replaceCommand([]uint8{0xFF, 0xE0, 0xDF, 0x3C, 0x3B})
			continue

		case ModAddHooks:
			if hookImpossible {
				return nil, nil, errors.New("mod conflict (ModNoDebug/ModAddHooks)")
			}
			hookImpossible = true

			var err error
			code, err = patchInstall(code, codeOffset, hooks)
			if err != nil {
				return nil, nil, err
			}
			continue

		default:
			return nil, nil, fmt.Errorf("unknown mod '%s'", m)
		}

		return nil, nil, notSupported(h, m)
	}

	return code, nvram, nil
}

func PatchCreate(fw []byte, mods []Mod) ([]byte, error) {
	if len(mods) == 0 {
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

	return image.Build(code, nvram, isRam), nil
}
