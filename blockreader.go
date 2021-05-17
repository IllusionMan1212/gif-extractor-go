package main

import "io"

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func readByte(r io.Reader) (byte, error) {
	var buf [1]byte
	n, err := r.Read(buf[:])
	if n == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return buf[0], err
}

func (v *blockReader) readNextBlock() error {
	blockSize, err := readByte(v.r)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return err
	}
	if blockSize == 0 {
		return io.EOF
	}
	_, err = io.ReadFull(v.r, v.buf[:blockSize])
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return err
	}
	v.bufLen = int(blockSize)
	v.bufNext = 0
	return nil
}

func (v *blockReader) Read(p []byte) (n int, err error) {
	if v.bufNext >= v.bufLen {
		err = v.readNextBlock()
		if err != nil {
			return 0, err
		}
	}
	n = min(len(p), v.bufLen-v.bufNext)
	for i := 0; i < n; i++ {
		p[i] = v.buf[i+v.bufNext]
	}
	v.bufNext += n
	return
}

type blockReader struct {
	buf     [255]byte
	bufLen  int
	bufNext int
	r       io.Reader
}

func newBlockReader(r io.Reader) *blockReader {
	return &blockReader{
		r:       r,
		bufLen:  0,
		bufNext: 0,
	}
}
