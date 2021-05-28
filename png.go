package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io/fs"
	"os"
)

func writeHeader(pngFile *os.File) {
	header := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	_, err := pngFile.Write(header)
	if err != nil {
		panic(err)
	}
}

func writeIHDR(pngFile *os.File, width uint16, height uint16) {
	chunkLength := []byte{0x00, 0x00, 0x00, 0x0D} // always 13
	IHDR := []byte{0x49, 0x48, 0x44, 0x52}        // IHDR string
	bitDepth := byte(0x8)                         // number of bits per sample or per palette index (always 8 for indexed-color)
	colorType := byte(0x3)                        // 3 is Indexed-color which is what's used for GIFs. this also means a PLTE chunk needs to exist
	compressionMethod := byte(0x0)                // always 0
	filterMethod := byte(0x0)                     // always 0
	interlaceMethod := byte(0x0)                  // 0 or 1
	hash := make([]byte, 4)                       // crc-32 hash of the previous 13 bytes

	chunkData := make([]byte, 13)

	binary.BigEndian.PutUint32(chunkData[0:4], uint32(width))
	binary.BigEndian.PutUint32(chunkData[4:8], uint32(height))
	chunkData[8] = bitDepth
	chunkData[9] = colorType
	chunkData[10] = compressionMethod
	chunkData[11] = filterMethod
	chunkData[12] = interlaceMethod

	binary.BigEndian.PutUint32(hash, crc32.Update(crc32.ChecksumIEEE(IHDR), crc32.IEEETable, chunkData))

	_, err := pngFile.Write(chunkLength)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(IHDR)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(chunkData)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(hash)
	if err != nil {
		panic(err)
	}
}

func writePLTE(pngFile *os.File, palette Palette) {
	chunkLength := make([]byte, 4)
	PLTE := []byte{0x50, 0x4C, 0x54, 0x45}
	hash := make([]byte, 4)

	binary.BigEndian.PutUint32(chunkLength, uint32(len(palette)*3))
	binary.BigEndian.PutUint32(hash, crc32.Update(crc32.ChecksumIEEE(PLTE), crc32.IEEETable, palette.MarshalBinary()))

	_, err := pngFile.Write(chunkLength)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(PLTE)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(palette.MarshalBinary())
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(hash)
	if err != nil {
		panic(err)
	}
}

func writetRNS(pngFile *os.File, transparencyIndex int) {
	chunkLength := make([]byte, 4)
	tRNS := []byte{0x74, 0x52, 0x4E, 0x53}
	hash := make([]byte, 4)

	binary.BigEndian.PutUint32(chunkLength, uint32(transparencyIndex)+1)

	data := make([]byte, transparencyIndex+1)
	for i := 0; i < transparencyIndex; i++ {
		data[i] = 0xFF
	}

	data[transparencyIndex] = 0

	binary.BigEndian.PutUint32(hash, crc32.Update(crc32.ChecksumIEEE(tRNS), crc32.IEEETable, data))

	_, err := pngFile.Write(chunkLength)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(tRNS)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(data)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(hash)
	if err != nil {
		panic(err)
	}
}

func serialize(data []byte, width int, height int) []byte {
	b := make([]byte, 0, (width+1)*height)
	for i := 0; i < height; i++ {
		b = append(b, 0)
		b = append(b, data[width*i:width*(i+1)]...)
	}
	return b
}

func writeIDAT(pngFile *os.File, data []byte, width uint16, height uint16) {
	chunkLength := make([]byte, 4)
	IDAT := []byte{0x49, 0x44, 0x41, 0x54}
	hash := make([]byte, 4)

	binary.BigEndian.PutUint32(chunkLength, uint32(len(data)))
	binary.BigEndian.PutUint32(hash, crc32.Update(crc32.ChecksumIEEE(IDAT), crc32.IEEETable, data))

	_, err := pngFile.Write(chunkLength)
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(IDAT)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		panic(err)
	}
	defer writer.Close()
	_, err = writer.Write(serialize(data, int(width), int(height)))
	if err != nil {
		panic(err)
	}
	writer.Flush()
	_, err = pngFile.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
	_, err = pngFile.Write(hash)
	if err != nil {
		panic(err)
	}
}

func writeIEND(pngFile *os.File) {
	chunkLength := []byte{0x00, 0x00, 0x00, 0x00}
	IEND := []byte{0x49, 0x45, 0x4E, 0x44}
	hash := make([]byte, 4)
	binary.BigEndian.PutUint32(hash, crc32.Update(crc32.ChecksumIEEE(IEND), crc32.IEEETable, nil))

	IENDChunk := append(chunkLength, append(IEND, hash...)...)
	_, err := pngFile.Write(IENDChunk)
	if err != nil {
		panic(err)
	}
}

func WriteToPNG(data []byte, palette Palette, fileName string, width uint16, height uint16, transparenyIndex int) {
	pngFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, fs.FileMode(0755))
	if err != nil {
		panic(err)
	}

	writeHeader(pngFile)
	writeIHDR(pngFile, width, height)
	writePLTE(pngFile, palette)
	if transparenyIndex != -1 {
		writetRNS(pngFile, transparenyIndex)
	}
	writeIDAT(pngFile, data, width, height)
	writeIEND(pngFile)
	pngFile.Close()
}
