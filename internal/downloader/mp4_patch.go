package downloader

import (
	"encoding/binary"
	"os"
	"time"
)

func patchMP4Times(path string, t time.Time) error {
	macEpoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	if t.Before(macEpoch) {
		return nil
	}
	sec1904 := uint64(t.Sub(macEpoch).Seconds())

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	fileSize := fi.Size()

	var walk func(offset int64, end int64) error
	walk = func(offset int64, end int64) error {
		for offset < end {
			if offset+8 > fileSize {
				break
			}
			var head [8]byte
			if _, err := f.ReadAt(head[:], offset); err != nil {
				return err
			}
			
			boxSize := uint64(binary.BigEndian.Uint32(head[0:4]))
			boxType := string(head[4:8])
			headerLen := int64(8)

			if boxSize == 1 {
				if offset+16 > fileSize {
					break
				}
				var ext [8]byte
				if _, err := f.ReadAt(ext[:], offset+8); err != nil {
					return err
				}
				boxSize = binary.BigEndian.Uint64(ext[:])
				headerLen = 16
			} else if boxSize == 0 {
				boxSize = uint64(end - offset)
			}

			if boxSize < uint64(headerLen) {
				break // Invalid box size, prevent infinite loop
			}
			
			boxEnd := offset + int64(boxSize)
			if boxEnd > end {
				boxEnd = end // Clamp to parent size
			}

			switch boxType {
			case "moov", "trak", "mdia":
				if err := walk(offset+headerLen, boxEnd); err != nil {
					return err
				}
			case "mvhd", "tkhd", "mdhd":
				if offset+headerLen+4 > fileSize {
					break
				}
				var ver [1]byte
				if _, err := f.ReadAt(ver[:], offset+headerLen); err != nil {
					return err
				}
				version := ver[0]

				if version == 0 {
					if offset+headerLen+12 <= fileSize {
						var times [8]byte
						binary.BigEndian.PutUint32(times[0:4], uint32(sec1904))
						binary.BigEndian.PutUint32(times[4:8], uint32(sec1904))
						_, _ = f.WriteAt(times[:], offset+headerLen+4)
					}
				} else if version == 1 {
					if offset+headerLen+20 <= fileSize {
						var times [16]byte
						binary.BigEndian.PutUint64(times[0:8], sec1904)
						binary.BigEndian.PutUint64(times[8:16], sec1904)
						_, _ = f.WriteAt(times[:], offset+headerLen+4)
					}
				}
			}
			
			offset = boxEnd
		}
		return nil
	}

	return walk(0, fileSize)
}
