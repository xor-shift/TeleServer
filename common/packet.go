package common

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"math/big"
)

type InnerPacket interface{}

type EssentialsPacket struct {
	Speed               float32    `json:"spd" mapstructure:"spd"`
	BatteryTemperatures [5]float32 `json:"temps" mapstructure:"temps"`
	Voltage             float32    `json:"v" mapstructure:"v"`
	RemainingWattHours  float32    `json:"wh" mapstructure:"wh"`
}

type FullPacket struct {
	BatteryVoltages     [27]float32 `json:"v" mapstructure:"v"`
	BatteryTemperatures [5]float32  `json:"temps" mapstructure:"temps"`
	SpentMilliAmpHours  float32     `json:"mah" mapstructure:"mah"`
	SpentMilliWattHours float32     `json:"mwh" mapstructure:"mwh"`
	Current             float32     `json:"amps" mapstructure:"amps"`
	PercentSOC          float32     `json:"soc" mapstructure:"soc"`

	HydroCurrent     float32 `json:"hc" mapstructure:"hc"`
	HydroPPM         float32 `json:"hd" mapstructure:"hd"`
	HydroTemperature float32 `json:"ht" mapstructure:"ht"`

	TemperatureSMPS         float32    `json:"ts" mapstructure:"ts"`
	TemperatureEngineDriver float32    `json:"ted" mapstructure:"ted"`
	VCEngineDriver          [2]float32 `json:"vced" mapstructure:"vced"`
	VCTelemetry             [2]float32 `json:"vct" mapstructure:"vct"`
	VCSMPS                  [2]float32 `json:"vcs" mapstructure:"vcs"`
	VCBMS                   [2]float32 `json:"vcb" mapstructure:"vcb"`

	Speed    float32    `json:"spd" mapstructure:"spd"`
	RPM      float32    `json:"rpm" mapstructure:"rpm"`
	VCEngine [2]float32 `json:"vce" mapstructure:"vce"`

	Longitude float32    `json:"long" mapstructure:"long"`
	Latitude  float32    `json:"lat" mapstructure:"lat"`
	Gyro      [3]float32 `json:"gyro" mapstructure:"gyro"`

	QueueFillAmount uint32  `json:"q" mapstructure:"q"`
	TickCounter     uint32  `json:"tc" mapstructure:"tc"`
	FreeHeap        uint32  `json:"heap" mapstructure:"heap"`
	AllocCount      uint32  `json:"alloc" mapstructure:"alloc"`
	FreeCount       uint32  `json:"free" mapstructure:"free"`
	CPUUsage        float32 `json:"cu" mapstructure:"cu"`
}

type PacketHeader struct {
	SequenceID uint   `json:"seq"`
	Timestamp  int32  `json:"ts"`
	RNGState   uint32 `json:"rng"`
}

type Packet struct {
	PacketHeader

	Inner InnerPacket `json:"data"`
}

type AMQPPacket struct {
	SessionID uint   `json:"sessionId"`
	Packet    Packet `json:"packet"`
}

func init() {
	gob.Register(EssentialsPacket{})
	gob.Register(FullPacket{})

	/*gob.RegisterName("EssentialsPacket", EssentialsPacket{})
	gob.RegisterName("FullPacket", FullPacket{})
	gob.RegisterName("PacketHeader", PacketHeader{})
	gob.RegisterName("Packet", Packet{})
	gob.RegisterName("AMQPPacket", AMQPPacket{})*/
}

func ParsePackets[T EssentialsPacket | FullPacket](body []byte, pubKey ecdsa.PublicKey) (packets []Packet, err error) {
	if len(body) < 128+2 {
		err = errors.New("body is too small even for an empty object and a signature")
		return
	}

	rBytes := body[len(body)-128 : len(body)-64]
	sBytes := body[len(body)-64 : len(body)]

	var r, s *big.Int
	var ok bool

	if r, ok = big.NewInt(0).SetString(string(rBytes), 16); !ok {
		err = errors.New("bad signature R")
		return
	}

	if s, ok = big.NewInt(0).SetString(string(sBytes), 16); !ok {
		err = errors.New("bad signature S")
		return
	}

	packets = []Packet{}

	jsonBody := body[:len(body)-128]
	jsonHash := sha256.Sum256(jsonBody)

	if !ecdsa.Verify(&pubKey, jsonHash[:], r, s) {
		err = errors.New("bad signature")
		return
	}

	if err = json.Unmarshal(jsonBody, &packets); err != nil {
		return
	}

	for k, v := range packets {
		/*_, ok := v.PacketData.(T)
		if !ok {
			err = errors.New(fmt.Sprintf("packet at index %d was not of the correct type", k))
			return
		}*/

		var packet T
		err = mapstructure.Decode(v.Inner, &packet)
		if err != nil {
			err = errors.New(fmt.Sprintf("packet at index %d was not of the correct type", k))
			return
		}

		packets[k].Inner = packet
	}

	return
}
