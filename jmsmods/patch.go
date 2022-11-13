package jmsmods

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"

	_ "embed"
)

type jumptableEntry struct {
	Type        uint8
	AddrEntry   uint16
	AddrHandler uint16
}

/* This will only patch the first jumptable, in the firmwares I have seen this one is always used first, so it should be enough */
func patchFindJumpTable(code []byte) ([]jumptableEntry, error) {
	/* Search for DF xx xx E0 xx xx FF 00 00 */
	evalIsTable := func(buf []byte) bool {
		return len(buf) >= 9 && buf[0] == 0xDF && buf[3] == 0xE0 && buf[6] == 0xFF && buf[7] == 0 && buf[8] == 0
	}

	/* Search for E0 | 12 xx xx | xx xx 03 */
	evalIsStart := func(buf []byte) bool {
		return len(buf) >= 7 && buf[0] == 0xe0 && buf[1] == 0x12 && buf[6] == 0x03
	}

	for i := range code {
		if evalIsTable(code[i:]) {
			/* Find start of table */

			for k := i; k >= 0; k-- {
				if evalIsStart(code[k:]) {
					k += 4

					/* Read all entries */
					var result []jumptableEntry
					for {
						if addr := binary.BigEndian.Uint16(code[k:]); addr == 0 {
							break
						} else {
							result = append(result, jumptableEntry{
								Type:        code[k+2],
								AddrEntry:   uint16(k),
								AddrHandler: addr,
							})
						}
						k += 3
					}

					return result, nil
				}
			}
		}
	}

	return nil, errors.New("SCSI jumptable not found")
}

type HookFunc struct {
	Binary   []byte
	Relocate func(in []byte, loadAddr uint16) []byte
}

//go:embed asm/hook.bin
var hookBinaryMain []byte

//go:embed asm/reset.bin
var HookBinaryReset []byte

//go:embed asm/spi_rx.bin
var hookSPIReceive []byte

//go:embed asm/spi_tx.bin
var hookSPITransmit []byte

var hooks = []HookFunc{
	{Binary: HookBinaryReset}, // USB disconnect and reset chip

	{Binary: hookSPIReceive},  // SPI DMA Receive
	{Binary: hookSPITransmit}, // SPI DMA Transmit
}

const HookVersion string = "00.00.05" //This 8-byte string must be updated whenever the definitions change incompatibly

func patchFindLoadAddress(code []byte) uint16 {
	patchLoadAddr := uint16(len(code) - 0x1a)
	for i := len(code) - 0x20; i >= 0; i-- {
		if code[i] != 0 && code[i] != 0xFF {
			break
		}

		patchLoadAddr = uint16(i)
	}
	return patchLoadAddr + 0x20
}

func patchInstall(code []byte, codeOffset uint16, hooks []HookFunc) ([]byte, error) {
	codeCpy := make([]byte, len(code))
	copy(codeCpy, code)
	code = codeCpy

	/* Find free address */
	patchLoadAddr := patchFindLoadAddress(code)

	patchInfoTable := make([]byte, 8, 128)
	copy(patchInfoTable, []byte(HookVersion))

	/* Write the individual functions */
	for _, m := range hooks {
		bin := m.Binary
		if m.Relocate != nil {
			bin = m.Relocate(bin, patchLoadAddr)
		}

		var addrBuf [2]byte
		binary.BigEndian.PutUint16(addrBuf[:], uint16(codeOffset+patchLoadAddr))
		patchInfoTable = append(patchInfoTable, addrBuf[:]...)

		n := copy(code[patchLoadAddr:], bin)
		patchLoadAddr += uint16(n)
	}

	/* Indicate last element */
	patchInfoTable = append(patchInfoTable, []byte{0, 0}...)

	if len(patchInfoTable) > 128 {
		return nil, errors.New("information table too big")
	}
	copy(code[patchLoadAddr:], patchInfoTable)
	patchInfoTableAddr := codeOffset + patchLoadAddr
	patchLoadAddr += uint16(len(patchInfoTable))

	patchJumpTable, err := patchFindJumpTable(code)
	if err != nil {
		return nil, err
	}

	hook := make([]byte, len(hookBinaryMain))
	copy(hook, hookBinaryMain)

	/* Write info table address */
	for i, m := range hook {
		if m == 0xAA {
			hook[i] = byte(patchInfoTableAddr >> 8)
		} else if m == 0xBB {
			hook[i] = byte(patchInfoTableAddr)
		}
	}

	var patchJumpAddr uint16
	for _, k := range patchJumpTable {
		if k.Type == 0xe0 {
			patchJumpAddr = k.AddrEntry
			break
		}
	}
	if patchJumpAddr == 0 {
		return nil, errors.New("failed to replace SCSI handler 0xe0: not found")
	}

	/* Replace handler function */
	var origHandler [2]byte
	copy(origHandler[:], code[patchJumpAddr:])
	binary.BigEndian.PutUint16(code[patchJumpAddr:], codeOffset+patchLoadAddr)

	/* Relocate hook LCALL */
	offset := binary.BigEndian.Uint16(hook[0xA:])
	offset += codeOffset + patchLoadAddr
	binary.BigEndian.PutUint16(hook[0xA:], offset)

	hook = bytes.Replace(hook, []byte{0xde, 0xad}, origHandler[:], 1)

	/* Put entrypoint into code */
	copy(code[patchLoadAddr:], hook)

	return code, nil
}

var knownROM = []byte{0xb9, 0xdf, 0xa8, 0x5d, 0x37, 0x55, 0x49, 0x2e, 0x76, 0xb8, 0x66, 0x49, 0x2f, 0x93, 0x7a, 0xb0, 0xba, 0x98, 0x38, 0x5b}

func PatchBootromForHAL(bootrom []byte) ([]byte, error) {
	if len(bootrom) != 0x4000 {
		return nil, errors.New("bootrom must be 16kB")
	}

	sum := sha1.Sum(bootrom)
	if !bytes.Equal(sum[:], knownROM) {
		return nil, errors.New("bootrom is not known to this library")
	}

	/* Stop the bootrom trying to load FW from flash */
	bootrom[0x11a7] = 0x22

	return patchInstall(bootrom, 0, hooks)
}

func PatchReadInfo(hookInfoTable [128]byte) ([]uint16, string) {
	work := hookInfoTable[8:]

	var hookAddrs []uint16
	for len(work) >= 2 {
		addr := binary.BigEndian.Uint16(work)
		work = work[2:]

		if addr == 0 {
			break
		}
		hookAddrs = append(hookAddrs, addr)
	}

	fwHookVersion := string(hookInfoTable[:8])

	if len(hookAddrs) == 0 {
		return nil, ""
	}

	if fwHookVersion != HookVersion {
		hookAddrs = hookAddrs[:1]
	}

	return hookAddrs, fwHookVersion
}
