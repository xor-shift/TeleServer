package ingest

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	amqp "github.com/streadway/amqp"
	"github.com/xor-shift/teleserver/common"
	"github.com/xor-shift/teleserver/util"
	"log"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sync"
)

func advanceRNG(s [4]uint32) uint32 {
	result := util.RotL(s[0]+s[3], 7) + s[0]

	t := s[1] << 9

	s[2] ^= s[0]
	s[3] ^= s[1]
	s[1] ^= s[2]
	s[0] ^= s[3]

	s[2] ^= t

	s[3] = util.RotL(s[3], 11)

	return result
}

type state struct {
	sessionID        uint
	initialRNGVector [4]uint32

	droppedPacketCt uint
}

// honestly this is a bad idea but xoroshiro is fairly fast
func (state *state) getNthRNG(n uint) uint32 {
	n += 1

	var result uint32

	s := state.initialRNGVector

	for i := uint(0); i < n; i++ {
		result = util.RotL(s[0]+s[3], 7) + s[0]

		t := s[1] << 9

		s[2] ^= s[0]
		s[3] ^= s[1]
		s[1] ^= s[2]
		s[0] ^= s[3]

		s[2] ^= t

		s[3] = util.RotL(s[3], 11)
	}

	return result
}

type Ingest struct {
	db       *sql.DB
	amqpConn *amqp.Connection
	pubKey   ecdsa.PublicKey

	resetToken [32]uint8

	state state

	packetProcessorWG *sync.WaitGroup
	incomingPackets   chan []common.Packet
}

func NewIngester(pubKey ecdsa.PublicKey) (*Ingest, error) {
	var err error

	ingester := &Ingest{
		db:       nil,
		amqpConn: nil,
		pubKey:   pubKey,

		resetToken: [32]uint8{},

		state: state{},

		packetProcessorWG: &sync.WaitGroup{},
		incomingPackets:   make(chan []common.Packet, 128),
	}

	ingester.resetResetToken()

	if ingester.amqpConn, err = amqp.Dial(os.Getenv("AMQP_URL")); err != nil {
		return nil, err
	}

	dbConfig := mysql.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASSWORD"),
		Addr:                 os.Getenv("DB_ADDRESS"),
		DBName:               os.Getenv("DB_NAME"),
		Collation:            "utf8mb4_general_ci",
		Net:                  "tcp",
		AllowNativePasswords: true,
	}

	if ingester.db, err = sql.Open("mysql", dbConfig.FormatDSN()); err != nil {
		return nil, err
	}

	return ingester, nil
}

func (ingest *Ingest) resetResetToken() {
	generateBytes := func(sz uint) []uint8 {
		if mod := sz % 8; mod != 0 {
			sz += 8 - mod
		}

		ret := make([]uint8, sz)

		for i := uint(0); i < sz/8; i++ {
			v := rand.Uint64()

			for j := uint(0); j < 8; j++ {
				ret[i*8+j] = uint8(v & 0xFF)
				v >>= 8
			}
		}

		return ret
	}

	generated := generateBytes(32)
	copy(ingest.resetToken[:], generated[:32])
}

func (ingest *Ingest) validateResetSignature(r, s *big.Int) error {
	if !ecdsa.Verify(&ingest.pubKey, ingest.resetToken[:], r, s) {
		return errors.New("invalid signature")
	}

	return nil
}

func (ingest *Ingest) validateStringResetSignature(r, s string) error {
	rInt, rOk := big.NewInt(0).SetString(r, 16)
	sInt, sOk := big.NewInt(0).SetString(s, 16)

	if !rOk {
		return errors.New("bad r value")
	}

	if !sOk {
		return errors.New("bad s value")
	}

	return ingest.validateResetSignature(rInt, sInt)
}

func (ingest *Ingest) GetResetChallenge() string {
	return fmt.Sprintf("%064s", big.NewInt(0).SetBytes(ingest.resetToken[:]).Text(16))
}

func (ingest *Ingest) ResetChallengeResponse(body string) error {
	if len(body) != 128 {
		return errors.New(fmt.Sprintf("bad body length (expected 128, got %d)", len(body)))
	}

	r := string(body[0:64])
	s := string(body[64:128])

	if err := ingest.validateStringResetSignature(r, s); err != nil {
		return err
	}

	ingest.state.sessionID = 0
	ingest.state.initialRNGVector = [4]uint32{rand.Uint32(), rand.Uint32(), rand.Uint32(), rand.Uint32()}
	//ingest.state.nextSequenceID = 0
	//ingest.state.rngVector = ingest.state.initialRNGVector

	rows, err := ingest.db.Query(
		"insert into sessions (prng, challenge, csig_r, csig_s) values (?, ?, ?, ?) returning session_id",
		util.ArrayToString(ingest.state.initialRNGVector[:]),
		util.ArrayToString(ingest.resetToken[:]),
		r, s)

	if err != nil {
		return err
	}

	defer rows.Close()

	if !rows.Next() {
		return errors.New("no rows returned from sql insert query")
	}

	if err := rows.Scan(&ingest.state.sessionID); err != nil {
		return err
	}

	ingest.resetResetToken()

	return nil
}

func (ingest *Ingest) SessionID() uint {
	return ingest.state.sessionID
}

func (ingest *Ingest) GetInitialRNGVector() string {
	return fmt.Sprintf("%08x%08x%08x%08x",
		ingest.state.initialRNGVector[0],
		ingest.state.initialRNGVector[1],
		ingest.state.initialRNGVector[2],
		ingest.state.initialRNGVector[3])
}

func (ingest *Ingest) NewPackets(packets []common.Packet) error {
	ingest.incomingPackets <- packets
	return nil
}

// Start starts a certain number of worker threads for incoming packet batches.
// If `numThreads` is greater than 1, the data in the database will be inserted out of order.
// Honestly just pass in 1, to be safe...
func (ingest *Ingest) Start(numThreads uint) {
	ingest.packetProcessorWG.Add(int(numThreads))

	for i := uint(0); i < numThreads; i++ {
		go ingest.task()
	}
}

func (ingest *Ingest) Stop() {
	close(ingest.incomingPackets)
	ingest.packetProcessorWG.Wait()
}

func (ingest *Ingest) newPacket(packet *common.Packet) error {
	/*if packet.SequenceID < ingest.state.nextSequenceID {
		return errors.New(fmt.Sprintf("old sequence ID (got: %d, expected (at least): %d)",
			packet.SequenceID,
			ingest.state.nextSequenceID))
	}*/

	//seqDelta := packet.SequenceID - ingest.state.nextSequenceID + 1
	snapshot := ingest.state

	//expectedRNG := uint32(0)
	//for i := uint(0); i < seqDelta; i++ {
	//	expectedRNG = ingest.state.advance()
	//}

	expectedRNG := ingest.state.getNthRNG(packet.SequenceID)

	if packet.RNGState != expectedRNG {
		ingest.state = snapshot

		return errors.New(fmt.Sprintf("bad pRNG state (!) (got: %d, expected: %d)", packet.RNGState, expectedRNG))
	}

	//currentUnix := int32(time.Now().In(time.UTC).Unix())
	//currentDelay := currentUnix - packet.Timestamp
	//state.delayAveragingWindow[state.delayWindowPointer%len(state.delayAveragingWindow)] = currentDelay
	//state.delayWindowPointer++

	//ingest.state.droppedPacketCt += seqDelta - 1
	//state.lastFullPacket = *packet

	//state.lastPacket = *packet
	if inner, ok := packet.Inner.(common.FullPacket); ok {
		minV := math.MaxFloat64
		maxV := -math.MaxFloat64
		sumV := float32(0)

		for _, v := range inner.BatteryVoltages {
			if v > 0.01 {
				minV = math.Min(minV, float64(v))
			}
			maxV = math.Max(maxV, float64(v))
			sumV += v
		}

		minC := math.MaxFloat64
		maxC := -math.MaxFloat64

		for _, v := range inner.BatteryTemperatures {
			if v > 0.01 {
				minC = math.Min(minC, float64(v))
			}
			maxC = math.Max(maxC, float64(v))
		}

		log.Printf("%d @ %f (fill: %d): %d (%d/%d), %f RPM, %f km/h, @ (%f, %f), %f/%f/%f V %f/%f°C (H: %f°C, %f ppm) %f A",
			packet.SequenceID,
			float32(inner.TickCounter)/1000.,
			inner.QueueFillAmount,
			inner.FreeHeap,
			inner.AllocCount,
			inner.FreeCount,
			inner.RPM,
			inner.Speed,
			inner.Longitude,
			inner.Latitude,
			minV,
			maxV,
			sumV,
			minC,
			maxC,
			inner.HydroTemperature,
			inner.HydroPPM,
			inner.Current,
		)
		//state.lastFullPacket = *packet
	}

	return nil
}

func (ingest *Ingest) processPacketBatch(batch []common.Packet, amqpChan *amqp.Channel, amqpExchange string) error {
	log.Printf("%d new packets", len(batch))

	for _, packet := range batch {
		/*marshalled, _ := json.Marshal(packet.PacketData)
		_, err := stmt.Exec(state.sessionNo, packet.SequenceID, packet.Timestamp, string(marshalled))*/

		var err error

		if err = ingest.newPacket(&packet); err != nil {
			return err
		}

		var marshalledPacket bytes.Buffer
		packetEncoder := gob.NewEncoder(&marshalledPacket)
		if err = packetEncoder.Encode(common.AMQPPacket{
			SessionID: ingest.SessionID(),
			Packet:    packet,
		}); err != nil {
			return err
		}

		if err = amqpChan.Publish(
			amqpExchange,
			"",
			true,
			false,
			amqp.Publishing{
				ContentType: "application/octet-stream",
				Body:        marshalledPacket.Bytes(),
			}); err != nil {
			return err
		}
	}

	return nil
}

func (ingest *Ingest) task() {
	defer ingest.packetProcessorWG.Done()

	var err error
	var amqpChan *amqp.Channel

	if amqpChan, err = ingest.amqpConn.Channel(); err != nil {
		log.Fatalf("Failed to establish an amqp channel: %s", err)
		return
	}

	defer amqpChan.Close()

	if err = amqpChan.ExchangeDeclare(
		"full_packets", // name
		"fanout",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	); err != nil {
		log.Fatalf("Failed to declare an amqp exchange: %s", err)
		return
	}

	for batch := range ingest.incomingPackets {
		if err := ingest.processPacketBatch(batch, amqpChan, "full_packets"); err != nil {
			log.Printf("Error while processing a batch of %d packets: %s", len(batch), err)
		}
	}
}
