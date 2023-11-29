package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

type C byte
type I1 uint8
type I2 uint16
type I4 uint32
type R4 float32
type R8 float64

const (
	LittleEndian = 0
	BigEndian    = 1
)

type HMFile struct {
	basicInfo BasicInformation
	dataInfo  DataInformationBlock
}

type Header struct {
	BlockNumber I1
	BlockLength I2
}

type BasicInformation struct {
	Header
	TotalHeaderBlocks    I2
	ByteOrder            I1
	Satellite            [16]C
	ProcessingCenter     [16]C
	ObservationArea      [4]C
	ObservationAreaInfo  [2]C
	ObservationTimeline  I2
	ObservationStartTime R8
	ObservationEndTime   R8
	FileCreationTime     R8
	TotalHeaderLength    I4
	TotalDataLength      I4
	QualityFlag1         I1
	QualityFlag2         I1
	QualityFlag3         I1
	QualityFlag4         I1
	FileFormatVersion    [32]C
	FileName             [128]C
	Spare                [40]C
}

type DataInformationBlock struct {
	Header
	NumberOfBitsPerPixel I2
	NumberOfColumns      I2
	NumberOfLines        I2
	CompressionFlag      I1
	Spare                [40]C
}

func main() {
	f, err := os.Open("sample-data/HS_H09_20231031_1340_B02_FLDK_R10_S0110.DAT")
	if err != nil {
		panic(err)
	}

	// Decode basic info
	// Detect byte order. I1+I2+I2=5
	_, err = f.Seek(5, 0)
	if err != nil {
		panic(err)
	}
	i := BasicInformation{}
	read(f, binary.BigEndian, &i.ByteOrder)
	var o binary.ByteOrder
	fmt.Println(i.ByteOrder)
	if i.ByteOrder == LittleEndian {
		fmt.Println("little")
		o = binary.LittleEndian
	} else {
		fmt.Println("big")
		o = binary.BigEndian
	}
	_, _ = f.Seek(0, 0)
	read(f, o, &i.BlockNumber)
	read(f, o, &i.BlockLength)
	read(f, o, &i.TotalHeaderBlocks)
	read(f, o, &i.ByteOrder)
	read(f, o, &i.Satellite)
	read(f, o, &i.ProcessingCenter)
	read(f, o, &i.ObservationArea)
	read(f, o, &i.ObservationAreaInfo)
	read(f, o, &i.ObservationTimeline)
	read(f, o, &i.ObservationStartTime)
	read(f, o, &i.ObservationEndTime)
	read(f, o, &i.FileCreationTime)
	read(f, o, &i.TotalHeaderLength)
	read(f, o, &i.TotalDataLength)
	read(f, o, &i.QualityFlag1)
	read(f, o, &i.QualityFlag2)
	read(f, o, &i.QualityFlag3)
	read(f, o, &i.QualityFlag4)
	read(f, o, &i.FileFormatVersion)
	read(f, o, &i.FileName)
	read(f, o, &i.Spare)

	// Decode data information block
	d := DataInformationBlock{}
	read(f, o, &d.BlockNumber)
	read(f, o, &d.BlockLength)
	read(f, o, &d.NumberOfBitsPerPixel)
	read(f, o, &d.NumberOfColumns)
	read(f, o, &d.NumberOfLines)
	read(f, o, &d.CompressionFlag)
	read(f, o, &d.Spare)

	h := HMFile{basicInfo: i, dataInfo: d}

	fmt.Printf("%+v\n", h)
}

// read util function that reads and ignore error
func read(f *os.File, o binary.ByteOrder, dst any) {
	_ = binary.Read(f, o, dst)
}
