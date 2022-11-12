package scsi

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"
)

func readVIDPID(file string) (uint16, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}

	if len(data) < 4 {
		return 0, errors.New("vid/pid entry is too short")
	}

	result, err := strconv.ParseUint(string(data[:4]), 16, 16)
	return uint16(result), err
}

func findBlockDeviceForHost(host int) (string, error) {
	blockdev := "/sys/block/"

	entries, err := os.ReadDir(blockdev)
	if err != nil {
		return "", err
	}

	for _, m := range entries {
		name := m.Name()

		dst, err := os.Readlink(path.Join(blockdev, name, "device"))
		if err != nil {
			continue
		}

		if !strings.HasPrefix(dst, "../../../") {
			continue
		}
		dst = dst[9:]

		if index := strings.Index(dst, ":"); index < 0 {
			continue
		} else {
			dst = dst[:index]
		}

		hostDev, err := strconv.ParseUint(dst, 10, 64)
		if err != nil {
			continue
		}

		if int(hostDev) == host {
			return "/dev/" + name, nil
		}
	}

	return "", errors.New("matching block device was not found")
}

func FindUSBDevices(vid uint16, pid uint16) ([]string, error) {
	scsi := "/sys/bus/scsi/devices"

	entries, err := os.ReadDir(scsi)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, m := range entries {
		name := m.Name()

		if !strings.HasPrefix(name, "host") {
			continue
		}
		dev := path.Join(scsi, name)

		/* Check if it is USB and has the right VID/PID */
		vendorID, err := readVIDPID(dev + "/../../idVendor")
		if err != nil {
			continue
		}
		productID, _ := readVIDPID(dev + "/../../idProduct")

		if (vid > 0 && vendorID != vid) || (pid > 0 && productID != pid) {
			continue
		}

		host, err := strconv.ParseUint(name[4:], 10, 64)
		if err != nil {
			continue
		}

		if dev, err := findBlockDeviceForHost(int(host)); err == nil {
			results = append(results, dev)
		}
	}

	return results, nil
}

func isUsbPath(path string) (uint16, uint16, bool) {
	if len(path) != 9 || path[4] != ':' {
		return 0, 0, false
	}

	vid, err := strconv.ParseUint(path[:4], 16, 16)
	if err != nil {
		return 0, 0, false
	}

	pid, err := strconv.ParseUint(path[5:], 16, 16)
	if err != nil {
		return 0, 0, false
	}

	return uint16(vid), uint16(pid), true
}
