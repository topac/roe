package roe

// bmpHeader represents the header fields needed to build a valid bmp image
type bmpHeader struct {
	FileType                [2]byte
	FileSize                uint32
	Reserved1               uint16
	Reserved2               uint16
	BitmapOffset            uint32
	SizeOfBitmapHeader      uint32
	PixelWidth              uint32
	PixelHeight             uint32
	Planes                  uint16
	BitsPerPixel            uint16
	Compression             uint32
	ImageSize               uint32
	HorizontalResolution    uint32
	VerticalResolution      uint32
	NumberOfColorsInPalette uint32
	ImportantColors         uint32
}

func newBmpHeader(width, height int) bmpHeader {
	header := bmpHeader{}

	var SizeOfData = uint32(4 * width * height)

	header.FileType = [2]byte{'B', 'M'}
	header.FileSize = 40 + 14 + SizeOfData
	header.BitmapOffset = 40 + 14
	header.SizeOfBitmapHeader = 40
	header.PixelWidth = uint32(width)
	header.PixelHeight = uint32(height)
	header.Compression = 0
	header.BitsPerPixel = 32
	header.Planes = 1
	header.ImageSize = SizeOfData
	header.NumberOfColorsInPalette = 0
	header.ImportantColors = 0

	return header
}
