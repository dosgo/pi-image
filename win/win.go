package win

import "encoding/binary"

// FSCTL_LOCK_VOLUME = CTL_CODE（FILE_DEVICE_FILE_SYSTEM，6，METHOD_BUFFERED，FILE_ANY_ACCESS）
const  FSCTL_LOCK_VOLUME  =  0x90018
const FSCTL_UNLOCK_VOLUME = 0x9001C;
const IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS = 0x00560000;


type diskExtent struct {
	DiskNumber     uint32
	StartingOffset uint64
	ExtentLength   uint64
}

type VolumeDiskExtents []byte

func (v *VolumeDiskExtents) Len() uint {
	return uint(binary.LittleEndian.Uint32([]byte(*v)))
}

func (v *VolumeDiskExtents) Extent(n uint) diskExtent {
	ba := []byte(*v)
	offset := 8 + 24*n
	return diskExtent{
		DiskNumber:     binary.LittleEndian.Uint32(ba[offset:]),
		StartingOffset: binary.LittleEndian.Uint64(ba[offset+8:]),
		ExtentLength:   binary.LittleEndian.Uint64(ba[offset+16:]),
	}
}