package spiflash

func completeIO(offset uint32, buf []byte, f func(offset uint32, buf []byte) (int, error)) (int, error) {
	index := 0

	for len(buf) > 0 {
		n, err := f(offset, buf)
		index += n
		offset += uint32(n)

		if err != nil {
			return index, err
		}

		buf = buf[n:]
	}

	return index, nil
}

func pageCrossLength(offset uint32, txfr uint32, pageSize uint32) int {
	mask := (pageSize - 1)
	return int(pageSize - offset&mask)
}
