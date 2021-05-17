package main

const (
	EXTENSION_BLOCK = 0x21

	GRAPHICS_CONTROL_BLOCK      = 0xF9
	GRAPHICS_CONTROL_BLOCK_SIZE = 0x04

	PLAINTEXT_BLOCK      = 0x01
	PLAINTEXT_BLOCK_SIZE = 0x0C

	APPLICATION_BLOCK      = 0xFF
	APPLICATION_BLOCK_SIZE = 0x0B

	COMMENT_BLOCK    = 0xFE
	IMAGE_DESCRIPTOR = 0x2C
	TRAILER          = 0x3B
)

/*
HeaderPacked {
	0-2: 	GlobalColorTableSize
	  3: 	ColorTableSortFlag   | Only valid under 89a, 87a always sets it to 0
	4-6:	ColorResolution
	  7:	GlobalColorTableFlag
}
*/

type Header struct {
	Signature [3]byte // "GIF"
	Version   [3]byte // "87a" or "89a"

	// Logical Screen Descriptor
	ScreenWidth     uint16
	ScreenHeight    uint16
	Packed          byte
	BackgroundColor byte // unused if GlobalColorTableFlag is unset
	AspectRatio     byte
}

// [OPTIONAL]
// Comes right after the logical screen descriptor
// Size of color table is always a power of 2, with a max of 256 entries in the table
// ColorTableEntries = 1 << ((Packed & 7) + 1)
// ColorTableSize = 3 * (1 << ((Packed & 7) + 1))
type RGB struct {
	Red   byte
	Green byte
	Blue  byte
}

type Palette []RGB

/*
ImageDescriptorPacked {
	0:   LocalColorTableFlag | this flag is set (1) if the image contains a local color table
	1:   InterlaceFlag       | this flag is set (1) if the image is interlaced
	2:   SortFlag            | this flag is set (1) if the color table is sorted by importance (frequency of occurrence). only available on 89a
	3-4: Reserved
	5-7: LocalColorTableEntrySize
}
*/

type ImageDescriptor struct {
	Left   int16  // X position of image
	Top    int16  // Y position of image
	Width  uint16 // width of image in pixels
	Height uint16 // height of image in pixels
	Packed byte   // image and color table data information
}

/*
GraphicsControlPacked {
	0:   TransparentColorFlag
	1:   UserInputFlag
	2-4: DisposalMethod
	5-7: Reserved
}
*/

/********************/
/* EXTENSION BLOCKS */
/********************/

// Only available on 89a and comes after the global color table and before the 1st image
type GraphicsControlBlock struct {
	Packed                byte  // method of graphics disposal to use
	DelayTime             int16 // delay to wait
	TransparentColorIndex byte  // transparent color index
}

type PlainTextBlock struct {
	TextGridLeft     int16 // X position of text grid in pixels
	TextGridTop      int16 // Y position of the text grid in pixels
	TextGridWidth    int16 // width of text grid in pixels
	TextGridHeight   int16 // height of text grid in pixels
	CellWidth        byte  // width of grid cell in pixels
	CellHeight       byte  // height of grid cell in pixels
	TextFgColorIndex byte  // text foreground color index value
	TextBgColorIndex byte  // text background color index value
	PlainTextData    *byte // the plaintext data
}

type ApplicationExtensionBlock struct {
	Identifier      [8]byte // application identifier
	AuthentCode     [3]byte // application authentication code
	ApplicationData *byte   // application data
}

type CommentExtensionBlock struct {
	CommentData *byte // comment data
}
