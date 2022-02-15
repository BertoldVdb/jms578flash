package spiflash

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

type SPIFunc func(out []byte, in []byte) error

type Flash struct {
	spi SPIFunc

	deviceID [4]byte
	device   flashDevice

	maxBytesPerTransaction int
}

func New(spi SPIFunc, maxBytesPerTransaction int) (*Flash, error) {
	f := &Flash{
		spi: spi,

		maxBytesPerTransaction: maxBytesPerTransaction,
	}

	if err := f.readDeviceID(); err != nil {
		if err := f.readDeviceID(); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *Flash) readDeviceID() error {
	if err := f.spi([]byte{0x9F}, f.deviceID[:]); err != nil {
		return err
	}

	t := binary.BigEndian.Uint32(f.deviceID[:])
	var ok bool
	f.device, ok = deviceLookup(t)
	if !ok {
		return fmt.Errorf("unsupported flash type: %08x", t)
	}

	return nil
}

func (f *Flash) DeviceID() [4]byte {
	return f.deviceID
}

func (f *Flash) writeEnable() error {
	return f.spi([]byte{0x6}, nil)
}

func (f *Flash) statusRead() (uint8, error) {
	var result [1]byte
	err := f.spi([]byte{0x5}, result[:])
	return result[0], err
}

func (f *Flash) waitIdle(maxDuration time.Duration) error {
	timeout := time.Now().Add(maxDuration)
	for time.Now().Before(timeout) {
		if status, err := f.statusRead(); err != nil {
			return err
		} else {
			if status&1 == 0 {
				if status&(1<<5) > 0 {
					return errors.New("program operation failed")
				}
				return nil
			}
		}
	}
	return errors.New("timeout")
}

func (f *Flash) EraseChip() error {
	if err := f.writeEnable(); err != nil {
		return err
	}

	if err := f.spi([]byte{f.device.opcodeChipErase}, nil); err != nil {
		return err
	}

	err := f.waitIdle(2 * time.Second)
	return err
}

func (f *Flash) ErasePage(address uint32) error {
	if err := f.writeEnable(); err != nil {
		return err
	}

	var cmd [4]byte
	binary.BigEndian.PutUint32(cmd[:], address)
	cmd[0] = f.device.opcodePageErase

	if err := f.spi(cmd[:], nil); err != nil {
		return err
	}

	err := f.waitIdle(2 * time.Second)
	return err
}

func (f *Flash) write(offset uint32, data []byte) (int, error) {
	/* Do not write over page boundary */
	maxLen := pageCrossLength(offset, uint32(len(data)), f.device.pageSize)
	if len(data) > maxLen {
		data = data[:maxLen]
	}

	/* Do not waste time writing large 0xFFFFFF blocks */
	skippedFront := 0
	for i, m := range data {
		if m != 0xFF {
			offset += uint32(i)
			skippedFront = i
			data = data[i:]
			break
		}
	}

	skippedEnd := 0
	for len(data) > 0 && data[len(data)-1] == 0xFF {
		data = data[:len(data)-1]
		skippedEnd++
	}
	if len(data) == 0 {
		return skippedFront + skippedEnd, nil
	}

	/* Ensure the transmission is not too long */
	if len(data)+4 > f.maxBytesPerTransaction {
		data = data[:f.maxBytesPerTransaction-4]
		skippedEnd = 0
	}

	tmpBuf := make([]byte, 4, 4+len(data))
	binary.BigEndian.PutUint32(tmpBuf, offset)
	tmpBuf[0] = 0x2

	tmpBuf = append(tmpBuf, data...)

	if err := f.writeEnable(); err != nil {
		return 0, err
	}

	if err := f.spi(tmpBuf, nil); err != nil {
		return 0, err
	}

	if err := f.waitIdle(time.Second); err != nil {
		return 0, err
	}

	return skippedFront + skippedEnd + len(data), nil
}

func (f *Flash) Write(offset uint32, data []byte) (int, error) {
	return completeIO(offset, data, f.write)
}

func (f *Flash) read(offset uint32, data []byte) (int, error) {
	if len(data)+4 > f.maxBytesPerTransaction {
		data = data[:f.maxBytesPerTransaction-4]
	}

	var out [4]byte
	binary.BigEndian.PutUint32(out[:], offset)
	out[0] = 0x3

	if err := f.spi(out[:], data); err != nil {
		return 0, err
	}

	return len(data), nil
}

func (f *Flash) Read(offset uint32, data []byte) (int, error) {
	return completeIO(offset, data, f.read)
}
