package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type ImageData struct {
	Width   int
	Height  int
	Pix     []ColorRGB
	Palette [256]ColorRGB
	HasPal  bool
}

type ColorRGB struct {
	R, G, B byte
}

type PCXHeader struct {
	Manufacturer byte
	Version      byte
	Encoding     byte
	BitsPerPixel byte
	XMin, YMin   uint16
	XMax, YMax   uint16
	HDpi, VDpi   uint16
	Colormap     [48]byte
	Reserved     byte
	NumPlanes    byte
	BytesPerLine uint16
	PaletteInfo  uint16
	HScreenSize  uint16
	VScreenSize  uint16
	Filler       [54]byte
}

const (
	PCXPaletteMarker = 0x0C
	RLEThreshold     = 192
	PCXPaletteSize   = 768
	PCXHeaderSize    = 128
	PCXPaletteOffset = 769
	BMPHeaderSize    = 54
	BMPPaletteSize   = 256
	BiRGB            = 0
)

// LoadPCX читает PCX-файл в структуру ImageData (TrueColor)
func LoadPCX(filename string) (*ImageData, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var hdr PCXHeader
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}
	w := int(hdr.XMax - hdr.XMin + 1)
	h := int(hdr.YMax - hdr.YMin + 1)

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(stat.Size()-PCXPaletteOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	mark := make([]byte, 1)
	if _, err := f.Read(mark); err != nil {
		return nil, err
	}
	var palette [256]ColorRGB
	if mark[0] == PCXPaletteMarker {
		palData := make([]byte, PCXPaletteSize)
		if _, err := f.Read(palData); err != nil {
			return nil, err
		}
		for i := 0; i < 256; i++ {
			palette[i] = ColorRGB{
				R: palData[i*3+0],
				G: palData[i*3+1],
				B: palData[i*3+2],
			}
		}
	} else {
		for i := 0; i < 16; i++ {
			palette[i] = ColorRGB{
				R: hdr.Colormap[i*3+0],
				G: hdr.Colormap[i*3+1],
				B: hdr.Colormap[i*3+2],
			}
		}
	}
	_, err = f.Seek(PCXHeaderSize, io.SeekStart)
	if err != nil {
		return nil, err
	}
	imgData := make([]ColorRGB, w*h)
	bytesPerLine := int(hdr.BytesPerLine)
	var x, y int
	for y < h {
		b := make([]byte, 1)
		if _, err := f.Read(b); err != nil {
			break
		}
		if b[0] >= RLEThreshold {
			count := b[0] & 0x3F
			c := make([]byte, 1)
			if _, err := f.Read(c); err != nil {
				break
			}
			for j := 0; j < int(count) && x < bytesPerLine; j++ {
				if x < w {
					imgData[y*w+x] = palette[c[0]]
				}
				x++
			}
		} else {
			if x < w {
				imgData[y*w+x] = palette[b[0]]
			}
			x++
		}
		if x >= bytesPerLine {
			x = 0
			y++
		}
	}
	return &ImageData{Width: w, Height: h, Pix: imgData}, nil
}

// LoadBMP для отображения (уже готовый 8/24-бит BMP)
func LoadBMP(filename string) (*ImageData, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var fileHeader [14]byte
	if _, err := f.Read(fileHeader[:]); err != nil {
		return nil, err
	}
	if fileHeader[0] != 'B' || fileHeader[1] != 'M' {
		return nil, fmt.Errorf("не BMP")
	}
	var dibHeader [40]byte
	if _, err := f.Read(dibHeader[:]); err != nil {
		return nil, err
	}
	w := int(int32(binary.LittleEndian.Uint32(dibHeader[4:])))
	h := int(int32(binary.LittleEndian.Uint32(dibHeader[8:])))
	bitCount := binary.LittleEndian.Uint16(dibHeader[14:])

	var palette [256]ColorRGB
	var palCount int
	switch bitCount {
	case 8:
		palCount = 256
	case 24:
		palCount = 0
	default:
		return nil, fmt.Errorf("поддерживаются только 8 или 24 бита: %d", bitCount)
	}
	if palCount > 0 {
		for i := 0; i < palCount; i++ {
			var rgba [4]byte
			if err := binary.Read(f, binary.LittleEndian, &rgba); err != nil {
				return nil, err
			}
			palette[i] = ColorRGB{R: rgba[2], G: rgba[1], B: rgba[0]}
		}
	}
	rowSize := (w*int(bitCount)/8 + 3) &^ 3
	data := make([]byte, rowSize*h)
	if _, err := f.Read(data); err != nil {
		return nil, err
	}
	pix := make([]ColorRGB, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcY := h - 1 - y
			off := srcY * rowSize
			if bitCount == 8 {
				idx := data[off+x]
				c := palette[idx]
				pix[y*w+x] = c
			} else { // 24
				i := off + x*3
				b := data[i+0]
				g := data[i+1]
				r := data[i+2]
				pix[y*w+x] = ColorRGB{R: r, G: g, B: b}
			}
		}
	}
	return &ImageData{Width: w, Height: h, Pix: pix}, nil
}

// SaveBMP сохраняет 8-бит или 24-бит BMP (здесь 8 бит).
func SaveBMP(filename string, img *ImageData) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := img.Width
	h := img.Height
	bitCount := uint16(8)
	rowSize := (w + 3) &^ 3
	dataSize := rowSize * h
	fileSize := BMPHeaderSize + BMPPaletteSize*4 + dataSize

	// Заголовок файла
	var fileHeader [14]byte
	fileHeader[0] = 'B'
	fileHeader[1] = 'M'
	binary.LittleEndian.PutUint32(fileHeader[2:], uint32(fileSize))
	binary.LittleEndian.PutUint32(fileHeader[10:], BMPHeaderSize+BMPPaletteSize*4)

	// DIB-заголовок
	var dibHeader [40]byte
	binary.LittleEndian.PutUint32(dibHeader[0:], 40)
	binary.LittleEndian.PutUint32(dibHeader[4:], uint32(w))
	binary.LittleEndian.PutUint32(dibHeader[8:], uint32(h))
	binary.LittleEndian.PutUint16(dibHeader[12:], 1)
	binary.LittleEndian.PutUint16(dibHeader[14:], bitCount)
	binary.LittleEndian.PutUint32(dibHeader[16:], BiRGB)
	binary.LittleEndian.PutUint32(dibHeader[20:], uint32(dataSize))

	if _, err := f.Write(fileHeader[:]); err != nil {
		return err
	}
	if _, err := f.Write(dibHeader[:]); err != nil {
		return err
	}
	// Записываем палитру (256 * 4 байта: B,G,R,0) и пиксели
	for i := 0; i < 256; i++ {
		c := img.Palette[i]
		out := []byte{c.B, c.G, c.R, 0}
		if _, err := f.Write(out); err != nil {
			return err
		}
	}
	line := make([]byte, rowSize)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			idx := img.Pix[y*w+x].R
			line[x] = idx
		}
		if _, err := f.Write(line); err != nil {
			return err
		}
	}
	return nil
}
