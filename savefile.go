package main

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
)

const (
	slotHashOffset = 0x300
	slotDataOffset = 0x310
	slotDataLength = 0x280000
	slotStride     = 0x280010
)

type SaveFile struct {
	Path string
	Raw  []byte
}

func LoadSaveFile(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 || string(data[:4]) != "BND4" {
		return nil, fmt.Errorf("file is not a valid .sl2 file (missing BND4)")
	}

	_ = os.WriteFile(path+".bak", data, 0644)
	return &SaveFile{Path: path, Raw: data}, nil
}

func (s *SaveFile) GetSlotData(slot int) ([]byte, error) {
	if slot < 0 || slot > 9 {
		return nil, fmt.Errorf("invalid slot: %d", slot)
	}
	start := slotDataOffset + (slot * slotStride)
	if start+slotDataLength > len(s.Raw) {
		return nil, fmt.Errorf("slot is out of file bounds")
	}
	return s.Raw[start : start+slotDataLength], nil
}

func (s *SaveFile) Save() error {
	return os.WriteFile(s.Path, s.Raw, 0644)
}

func (s *SaveFile) UpdateSlotChecksum(slot int) {
	start := slotDataOffset + (slot * slotStride)
	hashStart := slotHashOffset + (slot * slotStride)
	hash := md5.Sum(s.Raw[start : start+slotDataLength])
	copy(s.Raw[hashStart:hashStart+16], hash[:])
}

func decodeUTF16LE(b []byte) string {
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}
