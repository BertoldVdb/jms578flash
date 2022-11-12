package jmshal

import (
	"encoding/binary"
	"errors"

	_ "embed"

	"github.com/BertoldVdb/jms578flash/jmsmods"
)

func (d *JMSHal) PatchIsCurrent() bool {
	return d.hookVersion == jmsmods.HookVersion
}

func (d *JMSHal) PatchVersion() (string, string) {
	return d.hookVersion, jmsmods.HookVersion
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

	d.hooks, d.hookVersion = jmsmods.PatchReadInfo(hookInfoTable)

	return d.spiInit()
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
