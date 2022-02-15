package jmshal

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"

	_ "embed"
)

func patchFindJumpTable(code []byte) (uint16, error) {
	/* Search for DF xx xx E0 xx xx FF 00 00 */
	eval := func(buf []byte) bool {
		return len(buf) >= 9 && buf[0] == 0xDF && buf[3] == 0xE0 && buf[6] == 0xFF && buf[7] == 0 && buf[8] == 0
	}

	for i := range code {
		if eval(code[i:]) {
			return uint16(i + 1), nil
		}
	}

	return 0, errors.New("SCSI jumptable not found")
}

type HookFunc struct {
	Binary   []byte
	Relocate func(in []byte, loadAddr uint16) []byte
}

//go:embed asm/hook.bin
var hookBinaryMain []byte

//go:embed asm/reset.bin
var hookBinaryReset []byte

//go:embed asm/spi_rx.bin
var hookSPIReceive []byte

//go:embed asm/spi_tx.bin
var hookSPITransmit []byte

var hooks = []HookFunc{
	{Binary: hookBinaryReset}, // USB disconnect and reset chip

	{Binary: hookSPIReceive},  // SPI DMA Receive
	{Binary: hookSPITransmit}, // SPI DMA Transmit
}

const libHookVersion string = "00.00.05" //This 8-byte string must be updated whenever the definitions change incompatibly

func patchInstall(code []byte, codeOffset uint16, hooks []HookFunc) ([]byte, error) {
	codeCpy := make([]byte, len(code))
	copy(codeCpy, code)
	code = codeCpy

	/* Find lowest free address */
	patchLoadAddr := uint16(len(code) - 0x1a)
	for i := len(code) - 0x20; i >= 0; i-- {
		if code[i] != 0 && code[i] != 0xFF {
			break
		}

		patchLoadAddr = uint16(i)
	}
	patchLoadAddr += 0x20

	patchInfoTable := make([]byte, 8, 128)
	copy(patchInfoTable, []byte(libHookVersion))

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

	patchJumpAddr, err := patchFindJumpTable(code)
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

func patchBootromForHAL(bootrom []byte) ([]byte, error) {
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

func (d *JMSHal) PatchIsPresent() bool {
	return d.hookVersion == libHookVersion
}

func (d *JMSHal) PatchVersion() (string, string) {
	return d.hookVersion, libHookVersion
}

func (d *JMSHal) hookUpdateAvailable() error {
	var cmdBuf [2]byte
	cmdBuf[0] = 0xe0
	cmdBuf[1] = 0x78

	var result [9]byte
	resultSlice := result[:]

	d.hooks = nil
	d.hookVersion = ""

	if err := d.dev.Read(cmdBuf[:], &resultSlice); err != nil {
		return nil
	}

	hookInfoTableAddr := binary.BigEndian.Uint16(result[:])
	var hookInfoTable [128]byte

	if _, err := d.CodeRead(hookInfoTableAddr, hookInfoTable[:]); err != nil {
		return err
	}

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
		return errors.New("invalid hook descriptor received")
	}

	if fwHookVersion != libHookVersion {
		hookAddrs = hookAddrs[:1]
	}

	d.hookVersion = fwHookVersion
	d.hooks = hookAddrs

	//TODO: Init SPI (do this in a better place)
	d.CodeCall(0x2c32, CPUContext{})

	return nil
}

type CPUContext struct {
	DPTR uint16
	ACC  uint8
	R    [8]uint8
}

func (d *JMSHal) CodeCall(addr uint16, ctx CPUContext) (CPUContext, error) {
	var cmdBuf [15]byte
	cmdBuf[0] = 0xe0
	cmdBuf[1] = 0x77

	binary.LittleEndian.PutUint16(cmdBuf[2:], addr)
	binary.LittleEndian.PutUint16(cmdBuf[4:], ctx.DPTR)
	copy(cmdBuf[6:], ctx.R[:])
	cmdBuf[6+8] = ctx.ACC

	ctx = CPUContext{}

	var result [9]byte
	resultSlice := result[:]

	if err := d.dev.Read(cmdBuf[:], &resultSlice); err != nil {
		return ctx, err
	}

	ctx.ACC = result[0]
	copy(ctx.R[:], result[1:])

	return ctx, nil
}

func (d *JMSHal) hookCallIndex(index int, ctx CPUContext) (CPUContext, error) {
	if len(d.hooks) <= index {
		return ctx, errors.New("function is not available")
	}

	return d.CodeCall(d.hooks[index], ctx)
}
