package main

import (
	"bytes"
	"compress/lzw"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

func readNextBytes(file *os.File) byte {
	nextBytes := make([]byte, 1)
	file.Read(nextBytes)
	file.Seek(-1, os.SEEK_CUR)

	return nextBytes[0]
}

func (v Palette) UnmarshalBinary(data []byte) error {
	if len(v)*3 != len(data) {
		return fmt.Errorf("len is not valid. required: %d, actual: %d", len(v)*3, len(data))
	}
	for i := 0; i < len(v); i++ {
		v[i].Red = data[i*3]
		v[i].Green = data[i*3+1]
		v[i].Blue = data[i*3+2]
	}
	return nil
}

func main() {
	inputFile := os.Args[1]

	file, err := os.OpenFile(inputFile, os.O_RDONLY, os.FileMode(0755))
	if err != nil {
		panic(err)
	}

	header := Header{}
	headerData := make([]byte, 13)
	file.Read(headerData)
	binary.Read(bytes.NewBuffer(headerData), binary.LittleEndian, &header)

	if strings.ToUpper(string(header.Signature[:])) != "GIF" {
		panic("Not a valid gif file")
	}

	fmt.Print("Beginning extraction of gif frames\n\n")

	fmt.Printf("GIF version is: %v\n", string(header.Version[:]))
	if string(header.Version[:]) == "89a" {
		fmt.Print("WARNING: some features are unimplemented for this version. Some images may not extract correctly\n\n")
	}
	fmt.Printf("GIF aspect ratio is: %v\n", header.AspectRatio)
	fmt.Printf("GIF background color is: %v\n", header.BackgroundColor)
	fmt.Printf("GIF height is: %v\n", header.ScreenHeight)
	fmt.Printf("GIF width is: %v\n", header.ScreenWidth)

	GlobalColorTableFlag := (header.Packed & 0x80) >> 7
	GlobalColorTableEntries := 1 << ((header.Packed & 7) + 1)
	GlobalColorTableSize := 3 * GlobalColorTableEntries // number of entries * 3 colors (r, g, b)

	fmt.Printf("\n")
	fmt.Printf("Global color table flag: %v\n", GlobalColorTableFlag)
	fmt.Printf("Global color table size: %v\n", GlobalColorTableSize)
	fmt.Printf("global color table entries: %v\n", GlobalColorTableEntries)

	st, err := file.Stat()
	if err != nil {
		panic(err)
	}
	dirName := strings.Split(st.Name(), ".gif")[0]

	fmt.Printf("%v\n", dirName)
	os.Mkdir(dirName, os.FileMode(0755))

	var palette Palette

	if GlobalColorTableFlag == 0 { // no global color table, use default one (???)
		// TODO: do we skip it. no idea if this is accurate
		file.Seek(int64(GlobalColorTableSize), os.SEEK_CUR)
	} else { // global color table exists, use it
		globalColorTable := make([]byte, GlobalColorTableSize)
		file.Read(globalColorTable)

		palette = make(Palette, GlobalColorTableEntries)
		palette.UnmarshalBinary(globalColorTable)
	}

	for i := 1; readNextBytes(file) != TRAILER; i++ {
		nextByte := make([]byte, 1)
		file.Read(nextByte)

		fmt.Printf("next byte is: 0x%x\n", nextByte[0])

		switch nextByte[0] {
		case EXTENSION_BLOCK:
			{
				nextSecondByte := make([]byte, 1)
				file.Read(nextSecondByte)

				fmt.Printf("next second byte is: 0x%x\n", nextSecondByte[0])

				switch nextSecondByte[0] {
				case PLAINTEXT_BLOCK:
					{
						file.Seek(PLAINTEXT_BLOCK_SIZE+1, os.SEEK_CUR)
						plaintextDataSize := make([]byte, 1)
						file.Read(plaintextDataSize)
						file.Seek(int64(plaintextDataSize[0]), os.SEEK_CUR)

						// TODO: read plaintext data and draw it to the frame (somehow idk ???)

						break
					}
				case GRAPHICS_CONTROL_BLOCK:
					{
						file.Seek(GRAPHICS_CONTROL_BLOCK_SIZE+1, os.SEEK_CUR)

						// TODO: read graphics control block and do something with it ???

						break
					}
				case APPLICATION_BLOCK:
					{
						file.Seek(APPLICATION_BLOCK_SIZE+1, os.SEEK_CUR)
						applicationDataSize := make([]byte, 1)
						file.Read(applicationDataSize)
						file.Seek(int64(applicationDataSize[0]), os.SEEK_CUR)

						// TODO: read application data and do something with it ???

						break
					}
				case COMMENT_BLOCK:
					{
						commentDataSize := make([]byte, 1)
						file.Read(commentDataSize)
						file.Seek(int64(commentDataSize[0]), os.SEEK_CUR)

						// TODO: read comment data and output it to file ???

						break
					}
				}
				break
			}
		case IMAGE_DESCRIPTOR:
			{
				imageDescriptor := ImageDescriptor{}
				imageDescriptorData := make([]byte, 9)
				file.Read(imageDescriptorData)
				binary.Read(bytes.NewBuffer(imageDescriptorData), binary.LittleEndian, &imageDescriptor)

				decompressedFrameSize := uint64(int64(imageDescriptor.Height) * int64(imageDescriptor.Width))

				LocalColorTableFlag := imageDescriptor.Packed & 1
				InterlaceFlag := (imageDescriptor.Packed & 2) >> 1
				SortFlag := (imageDescriptor.Packed & 4) >> 2
				LocalColorTableSize := (imageDescriptor.Packed & 0xE0) >> 5

				fmt.Printf("Local Color Table Flag: %v\n", LocalColorTableFlag)
				fmt.Printf("Interlace Flag: %v\n", InterlaceFlag)
				fmt.Printf("Sort Flag: %v\n", SortFlag)
				fmt.Printf("Local Color Table Size: %v\n", LocalColorTableSize)

				// TODO: implement deinterlacing
				// TODO: implement sort flag thingy ???
				// TODO: implement local color table
				// TODO: use color tables for correct colors on pixels
				// TODO: write the image data to PNG

				LZWCodeSize := make([]byte, 1)
				file.Read(LZWCodeSize)

				reader := lzw.NewReader(newBlockReader(file), lzw.LSB, int(LZWCodeSize[0]))
				defer reader.Close()

				decompressedFrameData := make([]byte, decompressedFrameSize)

				_, err = io.ReadFull(reader, decompressedFrameData)
				if err != nil {
					panic(err)
				}

				outFile, err := os.OpenFile(fmt.Sprintf("./%s/%s-raw.%v", dirName, st.Name(), i), os.O_CREATE|os.O_RDWR, os.FileMode(0755))
				if err != nil {
					panic(err)
				}

				outFile.Write(decompressedFrameData)

				break
			}
		}

		// skip the terminator byte
		file.Seek(1, os.SEEK_CUR)
	}

	fmt.Print("Extracted frames from gif successfully!\n")

	file.Close()
}
