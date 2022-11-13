package scsi

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	SG_DXFER_NONE        = -1
	SG_DXFER_TO_DEV      = -2
	SG_DXFER_FROM_DEV    = -3
	SG_DXFER_TO_FROM_DEV = -4

	SG_INFO_OK_MASK = 0x1
	SG_INFO_OK      = 0x0

	SG_IO = 0x2285
)

type SGIOHdr struct {
	InterfaceID    int32   // 'S' for SCSI generic (required)
	DxferDirection int32   // data transfer direction
	CmdLen         uint8   // SCSI command length (<= 16 bytes)
	MxSbLen        uint8   // max length to write to sbp
	IovecCount     uint16  // 0 implies no scatter gather
	DxferLen       uint32  // byte count of data transfer
	DxferP         uintptr // points to data transfer memory or scatter gather list
	CmdP           uintptr // points to command to perform
	SbP            uintptr // points to sense_buffer memory
	Timeout        uint32  // MAX_UINT -> no timeout (unit: millisec)
	Flags          uint32  // 0 -> default, see SG_FLAG...
	PackID         int32   // unused internally (normally)
	UsrPtr         uintptr // unused internally
	Status         uint8   // SCSI status
	MaskedStatus   uint8   // shifted, masked scsi status
	MsgStatus      uint8   // messaging level data (optional)
	SbLenWr        uint8   // byte count actually written to sbp
	HostStatus     uint16  // errors from host adapter
	DriverStatus   uint16  // errors from software driver
	ResID          int32   // dxfer_len - actual_transferred
	Duration       uint32  // time taken by cmd (unit: millisec)
	Info           uint32  // auxiliary information
}

type SCSI struct {
	path    string
	fd      int
	Timeout uint32
}

func New(path string) (*SCSI, error) {
	s := &SCSI{
		path:    path,
		fd:      -1,
		Timeout: 3000,
	}

	err := s.open()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SCSI) open() error {
	path := s.path
	if vid, pid, ok := isUsbPath(path); ok {
		devs, err := FindUSBDevices(vid, pid)
		if err != nil {
			return err
		}
		if len(devs) == 0 {
			return errors.New("USB device not found")
		}
		if len(devs) > 1 {
			return errors.New("more than one USB device found")
		}

		path = devs[0]
	}

	var err error
	s.fd, err = unix.Open(path, unix.O_RDWR, 0600)

	return err
}

func (s *SCSI) Reopen() error {
	s.Close()
	time.Sleep(400 * time.Millisecond)

	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := s.open(); err == nil {
			return nil
		}
	}

	return s.open()
}

func (s *SCSI) Close() error {
	if s.fd < 0 {
		return nil
	}

	fd := s.fd
	s.fd = -1

	return unix.Close(fd)
}

func (s *SCSI) SGIO(hdr *SGIOHdr) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(s.fd), SG_IO, uintptr(unsafe.Pointer(hdr)))
	if errno != 0 {
		return errno
	}

	if hdr.Info&SG_INFO_OK_MASK != SG_INFO_OK {
		return fmt.Errorf("SCSI Status: %08x, Host Status: %08x, Driver Status: %08x", hdr.Status, hdr.HostStatus, hdr.DriverStatus)
	}

	return nil
}

func (s *SCSI) Read(cmd []byte, data *[]byte) error {
	senseBuf := make([]byte, 32)

	hdr := SGIOHdr{
		InterfaceID:    'S',
		SbP:            uintptr(unsafe.Pointer(&senseBuf[0])),
		Timeout:        s.Timeout,
		MxSbLen:        uint8(len(senseBuf)),
		DxferP:         uintptr(unsafe.Pointer(&(*data)[0])),
		DxferLen:       uint32(len(*data)),
		DxferDirection: SG_DXFER_FROM_DEV,

		CmdLen: uint8(len(cmd)),
		CmdP:   uintptr(unsafe.Pointer(&cmd[0])),
	}

	return s.SGIO(&hdr)
}

func (s *SCSI) Write(cmd []byte, data []byte) error {
	senseBuf := make([]byte, 32)

	hdr := SGIOHdr{
		InterfaceID:    'S',
		SbP:            uintptr(unsafe.Pointer(&senseBuf[0])),
		Timeout:        s.Timeout,
		MxSbLen:        uint8(len(senseBuf)),
		DxferDirection: SG_DXFER_TO_DEV,

		CmdLen: uint8(len(cmd)),
		CmdP:   uintptr(unsafe.Pointer(&cmd[0])),
	}

	if len(data) > 0 {
		hdr.DxferP = uintptr(unsafe.Pointer(&data[0]))
		hdr.DxferLen = uint32(len(data))
	}

	return s.SGIO(&hdr)
}
