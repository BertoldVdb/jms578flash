package image

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"
)

func TestCRC(t *testing.T) {
	result := crcCalculateBlock([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	correct := uint32(0x948b389d)

	if result != correct {
		t.Errorf("CRC Error: %08x!=%08x", result, correct)
	}
}

func testImage(t *testing.T, path string, isRam bool) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		t.Log(path, "not found, skipping test")
		return
	}

	c := make([]byte, len(buf))
	copy(c, buf)

	if err := Validate(buf, isRam); err != nil {
		t.Error("Valid image rejected:", err)
	}

	if err := Validate(buf[:1300], isRam); err != ErrorInvalidLength {
		t.Error("Image with invalid header:", err)
	}

	buf[4]++
	if err := Validate(buf, isRam); err != ErrorInvalidHeader {
		t.Error("Image with invalid header:", err)
	}
	buf[4]--

	buf[0x4000]++
	if err := Validate(buf, isRam); err != ErrorInvalidCRC {
		t.Error("Image with invalid crc:", err)
	}
	buf[0x4000]--

	if !bytes.Equal(c, buf) {
		t.Error("Buffer was modified during test")
	}
}

func TestImage(t *testing.T) {
	/* These are self made images that have been tested to work
	 * on real hardware */
	testImage(t, "test/image_flash.bin", false)
	testImage(t, "test/image_ram.bin", true)

	/* These are vendor images with unclear distribution terms.
	 * You can download them (for example) from HardKernel. */
	testImage(t, "test/image_flash_vendor.bin.nodist", false)
	testImage(t, "test/image_ram_vendor.bin.nodist", true)
}

func getRandomBuf(length int) []byte {
	out := make([]byte, length)
	rand.Read(out)
	return out
}

func testBuildExtract(t *testing.T, isRam bool) {
	var code, nvram []byte
	code = getRandomBuf(0xc000 - 8)
	if !isRam {
		nvram = getRandomBuf(0x200)
	}

	output := Build(code, nvram, isRam)

	code2, nvram2, isRam2, err := Extract(output)
	if err != nil {
		t.Error("Failed to extract data from image:", err)
		return
	}

	if isRam != isRam2 {
		t.Error("Wrong type generated")
	}

	if !bytes.Equal(code, code2) {
		t.Error("Regenerated code is not equal to the input")
	}

	if !bytes.Equal(nvram, nvram2) {
		t.Error("Regenerated nvram is not equal to the input")
	}
}

func TestBuildExtract(t *testing.T) {
	testBuildExtract(t, false)
	testBuildExtract(t, true)
}
