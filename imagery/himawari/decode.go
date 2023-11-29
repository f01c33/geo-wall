package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
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
	BasicInfo      BasicInformation
	DataInfo       DataInformationBlock
	ProjectionInfo ProjectionInformationBlock
	NavigationInfo NavigationInformationBlock
}

type Position struct {
	X R8
	Y R8
	Z R8
}

type BasicInformation struct {
	BlockNumber          I1
	BlockLength          I2
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
	BlockNumber          I1
	BlockLength          I2
	NumberOfBitsPerPixel I2
	NumberOfColumns      I2
	NumberOfLines        I2
	CompressionFlag      I1
	Spare                [40]C
}

type ProjectionInformationBlock struct {
	BlockNumber             I1
	BlockLength             I2
	SubLon                  R8
	CFAC                    I4
	LFAC                    I4
	COFF                    R4
	LOFF                    R4
	DistanceFromEarthCenter R8
	EarthEquatorialRadius   R8
	EarthPolarRadius        R8
	RatioDiff               R8
	RatioPolar              R8
	RatioEquatorial         R8
	SDCoefficient           R8
	ResamplingTypes         I2
	ResamplingSize          I2
	Spare                   [40]C
}

type NavigationInformationBlock struct {
	BlockNumber                  I1
	BlockLength                  I2
	NavigationTime               R8
	SSPLongitude                 R8
	SSPLatitude                  R8
	DistanceFromEarthToSatellite R8
	NadirLongitude               R8
	NadirLatitude                R8
	SunPosition                  Position
	MoonPosition                 Position
	Spare                        [40]C
}

// TODO: use only io.Reader without seek
func DecodeFile(f io.ReadSeeker) (*HMFile, error) {
	// Decode basic info
	// Detect byte order. I1+I2+I2=5
	_, err := f.Seek(5, 0)
	if err != nil {
		return nil, err
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

	// Decode projection information block
	p := ProjectionInformationBlock{}
	read(f, o, &p.BlockNumber)
	read(f, o, &p.BlockLength)
	read(f, o, &p.SubLon)
	read(f, o, &p.CFAC)
	read(f, o, &p.LFAC)
	read(f, o, &p.COFF)
	read(f, o, &p.LOFF)
	read(f, o, &p.DistanceFromEarthCenter)
	read(f, o, &p.EarthEquatorialRadius)
	read(f, o, &p.EarthPolarRadius)
	read(f, o, &p.RatioDiff)
	read(f, o, &p.RatioPolar)
	read(f, o, &p.RatioEquatorial)
	read(f, o, &p.SDCoefficient)
	read(f, o, &p.ResamplingTypes)
	read(f, o, &p.ResamplingSize)
	read(f, o, &d.Spare)

	// Decode navigation information block
	n := NavigationInformationBlock{}
	read(f, o, &n.BlockNumber)
	read(f, o, &n.BlockLength)
	read(f, o, &n.NavigationTime)
	read(f, o, &n.SSPLongitude)
	read(f, o, &n.SSPLatitude)
	read(f, o, &n.DistanceFromEarthToSatellite)
	read(f, o, &n.NadirLongitude)
	read(f, o, &n.NadirLatitude)
	read(f, o, &n.SunPosition.X)
	read(f, o, &n.SunPosition.Y)
	read(f, o, &n.SunPosition.Z)
	read(f, o, &n.MoonPosition.X)
	read(f, o, &n.MoonPosition.Y)
	read(f, o, &n.MoonPosition.Z)
	read(f, o, &n.Spare)

	return &HMFile{BasicInfo: i, DataInfo: d, ProjectionInfo: p, NavigationInfo: n}, nil
}

func main() {
	f, err := os.Open("sample-data/HS_H09_20231031_1340_B02_FLDK_R10_S0110.DAT")
	if err != nil {
		panic(err)
	}
	h, err := DecodeFile(f)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", h)
}

// read util function that reads and ignore error
func read(f io.Reader, o binary.ByteOrder, dst any) {
	_ = binary.Read(f, o, dst)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
