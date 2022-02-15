package jmshal

import (
	"github.com/BertoldVdb/jms578flash/scsi"
)

type JMSHal struct {
	dev *scsi.SCSI

	hooks       []uint16
	hookVersion string
}

func New(dev *scsi.SCSI) (*JMSHal, error) {
	d := &JMSHal{
		dev: dev,
	}

	err := d.hookUpdateAvailable()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *JMSHal) reopen() error {
	if err := d.dev.Reopen(); err != nil {
		return err
	}

	return d.hookUpdateAvailable()

}

func (d *JMSHal) ResetChip() error {
	/* Not every firmware supports a reset command */
	if err := d.dev.Write([]byte{0xff, 0x4, 0x26, 'J', 'M'}, nil); err == nil {
		return d.reopen()
	}

	/* Try to call our own reset function */
	if len(d.hooks) > 0 {
		d.hookCallIndex(0, CPUContext{})
		return d.reopen()
	}

	/* As a last resort, try to run the reset function as firmware... */
	d.CodeWrite(hookBinaryReset[5:], true)

	return d.reopen()
}
