package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/xor-shift/teleserver/util/rng"
	"log"
	"math/big"
	"math/rand"
	"os"
	"time"
)

const (
	bigInsertQuery = "INSERT INTO packets (session_id, packet_order, reported_time" +
		", battery_voltages, battery_temperatures, spent_mah, spent_mwh, curr, percent_soc" +
		", hydro_curr, hydro_ppm, hydro_temps" +
		", temperature_smps, temperature_engine_driver, voltage_engine_driver, current_engine_driver, voltage_telemetry, current_telemetry, voltage_smps, current_smps, voltage_bms, current_bms" +
		", speed, rpm, voltage_engine, current_engine" +
		", latitude, longitude, gyro_x, gyro_y, gyro_z" +
		", queue_fill_amt, tick_counter, free_heap, alloc_count, free_count, cpu_usage" +
		") VALUES (?, ?, FROM_UNIXTIME(?), " +
		"?, ?, ?, ?, ?, ?, " +
		"?, ?, ?, " +
		"?, ?, ?, ?, ?, ?, ?, ?, ?, ?, " +
		"?, ?, ?, ?, " +
		"?, ?, ?, ?, ?, " +
		"?, ?, ?, ?, ?, ?)"
)

type State struct {
	db             *sql.DB
	preparedInsert *sql.Stmt
	sessionStarted bool
	sessionNo      int

	resetToken [32]uint8

	initialRNGVector [4]uint32
	currentRNGVector [4]uint32
	receivedPacketCt uint
	droppedPacketCt  uint

	delayAveragingWindow [10]int32
	delayWindowPointer   int

	lastPacket     PacketWrapper
	lastFullPacket PacketWrapper
}

// StateSnapshot is for rollbacks that can be made during packet retrieval, if there's an inconsistency with the incoming packet.
type StateSnapshot struct {
	rngVector        [4]uint32
	receivedPacketCt uint
	droppedPacketCt  uint

	delayAveragingWindow [10]int32
	delayWindowPointer   int
}

func NewState() (*State, error) {
	dbConfig := mysql.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASSWORD"),
		Addr:                 os.Getenv("DB_ADDRESS"),
		DBName:               os.Getenv("DB_NAME"),
		Collation:            "utf8mb4_general_ci",
		Net:                  "tcp",
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		return nil, err
	}

	/*preparedInsert, err := db.Prepare("INSERT INTO packets (session_id, packet_order, reported_time, inner_data) VALUES (?, ?, FROM_UNIXTIME(?), ?)")
	if err != nil {
		return nil, err
	}*/

	state := &State{}
	state.db = db
	//state.preparedInsert = preparedInsert
	state.resetResetToken()

	/*rows, err := db.Query("select packet_order from essential_packets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var order int
		err := rows.Scan(&order)
		if err != nil {
			return nil, err
		}
		log.Println(order)
	}*/

	return state, nil
}

func (state *State) TakeSnapshot() StateSnapshot {
	return StateSnapshot{
		rngVector:            state.currentRNGVector,
		receivedPacketCt:     state.receivedPacketCt,
		droppedPacketCt:      state.droppedPacketCt,
		delayAveragingWindow: state.delayAveragingWindow,
		delayWindowPointer:   state.delayWindowPointer,
	}
}

func (state *State) LoadSnapshot(snapshot StateSnapshot) {
	state.currentRNGVector = snapshot.rngVector
	state.receivedPacketCt = snapshot.receivedPacketCt
	state.droppedPacketCt = snapshot.droppedPacketCt
	state.delayAveragingWindow = snapshot.delayAveragingWindow
	state.delayWindowPointer = snapshot.delayWindowPointer
}

func (state *State) resetResetToken() {
	vals := []uint64{rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()}

	state.resetToken[0] = uint8((vals[0] >> 56) & 0xFF)
	state.resetToken[1] = uint8((vals[0] >> 48) & 0xFF)
	state.resetToken[2] = uint8((vals[0] >> 40) & 0xFF)
	state.resetToken[3] = uint8((vals[0] >> 32) & 0xFF)
	state.resetToken[4] = uint8((vals[0] >> 24) & 0xFF)
	state.resetToken[5] = uint8((vals[0] >> 16) & 0xFF)
	state.resetToken[6] = uint8((vals[0] >> 8) & 0xFF)
	state.resetToken[7] = uint8((vals[0] >> 0) & 0xFF)

	state.resetToken[8] = uint8((vals[1] >> 56) & 0xFF)
	state.resetToken[9] = uint8((vals[1] >> 48) & 0xFF)
	state.resetToken[10] = uint8((vals[1] >> 40) & 0xFF)
	state.resetToken[11] = uint8((vals[1] >> 32) & 0xFF)
	state.resetToken[12] = uint8((vals[1] >> 24) & 0xFF)
	state.resetToken[13] = uint8((vals[1] >> 16) & 0xFF)
	state.resetToken[14] = uint8((vals[1] >> 8) & 0xFF)
	state.resetToken[15] = uint8((vals[1] >> 0) & 0xFF)

	state.resetToken[16] = uint8((vals[2] >> 56) & 0xFF)
	state.resetToken[17] = uint8((vals[2] >> 48) & 0xFF)
	state.resetToken[18] = uint8((vals[2] >> 40) & 0xFF)
	state.resetToken[19] = uint8((vals[2] >> 32) & 0xFF)
	state.resetToken[20] = uint8((vals[2] >> 24) & 0xFF)
	state.resetToken[21] = uint8((vals[2] >> 16) & 0xFF)
	state.resetToken[22] = uint8((vals[2] >> 8) & 0xFF)
	state.resetToken[23] = uint8((vals[2] >> 0) & 0xFF)

	state.resetToken[24] = uint8((vals[3] >> 56) & 0xFF)
	state.resetToken[25] = uint8((vals[3] >> 48) & 0xFF)
	state.resetToken[26] = uint8((vals[3] >> 40) & 0xFF)
	state.resetToken[27] = uint8((vals[3] >> 32) & 0xFF)
	state.resetToken[28] = uint8((vals[3] >> 24) & 0xFF)
	state.resetToken[29] = uint8((vals[3] >> 16) & 0xFF)
	state.resetToken[30] = uint8((vals[3] >> 8) & 0xFF)
	state.resetToken[31] = uint8((vals[3] >> 0) & 0xFF)
}

func (state *State) GetResetToken() [32]uint8 {
	return state.resetToken
}

func (state *State) checkTokenSignature(rStr, sStr string) bool {
	r, couldParse := big.NewInt(0).SetString(rStr, 16)
	if !couldParse {
		return false
	}

	s, couldParse := big.NewInt(0).SetString(sStr, 16)
	if !couldParse {
		return false
	}

	return ecdsa.Verify(&publicKey, state.resetToken[:], r, s)
}

func (state *State) Reset(rStr, sStr string) error {
	if ok := state.checkTokenSignature(rStr, sStr); !ok {
		return errors.New("signature does not sign the challenge")
	}

	state.initialRNGVector = [4]uint32{rand.Uint32(), rand.Uint32(), rand.Uint32(), rand.Uint32()}
	state.currentRNGVector = state.initialRNGVector
	state.receivedPacketCt = 0
	state.droppedPacketCt = 0

	state.delayAveragingWindow = [10]int32{}
	state.delayWindowPointer = 0

	rows, err := state.db.Query(
		"insert into sessions (prng, challenge, csig_r, csig_s) values (?, ?, ?, ?) returning session_id",
		ArrayToString(state.initialRNGVector[:]),
		ArrayToString(state.resetToken[:]),
		rStr, sStr)

	if err != nil {
		return err
	}

	defer rows.Close()

	if !rows.Next() {
		return errors.New("no rows returned from sql insert query")
	}

	if err := rows.Scan(&state.sessionNo); err != nil {
		return err
	}

	state.resetResetToken()

	return nil
}

func (state *State) NewPackets(ctx context.Context, packets []PacketWrapper) error {
	log.Printf("%d new packets", len(packets))
	tx, err := state.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	//stmt, err := tx.Prepare("INSERT INTO packets (session_id, packet_order, reported_time, inner_data) VALUES (?, ?, FROM_UNIXTIME(?), ?)")
	stmt, err := tx.Prepare(bigInsertQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, packet := range packets {
		/*marshalled, _ := json.Marshal(packet.PacketData)
		_, err := stmt.Exec(state.sessionNo, packet.SequenceID, packet.Timestamp, string(marshalled))*/

		if err := state.newPacket(&packet); err != nil {
			return err
		}

		inner := packet.PacketData.(FullPacket)

		batteryVoltages, _ := json.Marshal(inner.BatteryVoltages[:])
		batteryTemperatures, _ := json.Marshal(inner.BatteryTemperatures[:])
		hydroTemperatures, _ := json.Marshal(inner.HydroTemperatures[:])

		_, err := stmt.Exec(
			state.sessionNo, packet.SequenceID, packet.Timestamp,
			string(batteryVoltages), string(batteryTemperatures), inner.SpentMilliAmpHours, inner.SpentMilliWattHours, inner.Current, inner.PercentSOC,
			inner.HydroCurrent, inner.HydroPPM, string(hydroTemperatures),
			inner.TemperatureSMPS, inner.TemperatureEngineDriver, inner.VCEngineDriver[0], inner.VCEngineDriver[1], inner.VCTelemetry[0], inner.VCTelemetry[1], inner.VCSMPS[0], inner.VCSMPS[1], inner.VCBMS[0], inner.VCBMS[1],
			inner.Speed, inner.RPM, inner.VCEngine[0], inner.VCEngine[1],
			inner.Latitude, inner.Longitude, inner.Gyro[0], inner.Gyro[1], inner.Gyro[2],
			inner.QueueFillAmount, inner.TickCounter, inner.FreeHeap, inner.AllocCount, inner.FreeCount, inner.CPUUsage,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (state *State) newPacket(packet *PacketWrapper) error {
	if packet.SequenceID < state.receivedPacketCt {
		return errors.New(fmt.Sprintf("old sequence ID (got: %d, expected (at least): %d)", packet.SequenceID, state.receivedPacketCt))
	}

	seqDelta := packet.SequenceID - state.receivedPacketCt + 1
	snapshot := state.TakeSnapshot()

	expectedRNG := uint32(0)
	for i := uint(0); i < seqDelta; i++ {
		expectedRNG = state.AdvanceState()
	}

	if packet.RNGState != expectedRNG {
		state.LoadSnapshot(snapshot)

		return errors.New(fmt.Sprintf("bad pRNG state (!) (got: %d, expected: %d)", packet.RNGState, expectedRNG))
	}

	currentUnix := int32(time.Now().In(time.UTC).Unix())
	currentDelay := currentUnix - packet.Timestamp
	state.delayAveragingWindow[state.delayWindowPointer%len(state.delayAveragingWindow)] = currentDelay
	state.delayWindowPointer++

	state.droppedPacketCt += seqDelta - 1
	state.lastFullPacket = *packet

	state.lastPacket = *packet
	if inner, ok := packet.PacketData.(FullPacket); ok {
		log.Printf("%d (dropped: %d) @ %d: %d, %d (%d/%d)",
			packet.SequenceID,
			state.droppedPacketCt,
			packet.Timestamp,
			inner.QueueFillAmount,
			inner.FreeHeap,
			inner.AllocCount,
			inner.FreeCount)
		state.lastFullPacket = *packet
	}

	return nil
}

func (state *State) insertPacketToDB(packet PacketWrapper) {
	//inner := packet.PacketData.(FullPacket)
	defer func() { state.lastPacket = packet }()

	if inner, ok := packet.PacketData.(FullPacket); ok {
		log.Printf("%d (dropped: %d) @ %d: %d, %d (%d/%d)",
			packet.SequenceID,
			state.droppedPacketCt,
			packet.Timestamp,
			inner.QueueFillAmount,
			inner.FreeHeap,
			inner.AllocCount,
			inner.FreeCount)
		state.lastFullPacket = packet
	}

	marshalled, _ := json.Marshal(packet.PacketData)

	_, err := state.db.Query("INSERT INTO packets (session_id, packet_order, reported_time, inner_data) VALUES (?, ?, FROM_UNIXTIME(?), ?)",
		state.sessionNo, packet.SequenceID, packet.Timestamp, string(marshalled))

	/*batteryVoltages, _ := json.Marshal(inner.BatteryVoltages[:])
	batteryTemperatures, _ := json.Marshal(inner.BatteryTemperatures[:])
	hydroTemperatures, _ := json.Marshal(inner.HydroTemperatures[:])

	_, err := state.db.Query("INSERT INTO packets (session_id, packet_order, reported_time"+
		", battery_voltages, battery_temperatures, spent_mah, spent_mwh, curr, percent_soc"+
		", hydro_curr, hydro_ppm, hydro_temps"+
		", temperature_smps, temperature_engine_driver, voltage_engine_driver, current_engine_driver, voltage_telemetry, current_telemetry, voltage_smps, current_smps, voltage_bms, current_bms"+
		", speed, rpm, voltage_engine, current_engine"+
		", latitude, longitude, gyro_x, gyro_y, gyro_z"+
		", queue_fill_amt, tick_counter, free_heap, alloc_count, free_count, cpu_usage"+
		") VALUES (?, ?, FROM_UNIXTIME(?), "+
		"?, ?, ?, ?, ?, ?, "+
		"?, ?, ?, "+
		"?, ?, ?, ?, ?, ?, ?, ?, ?, ?, "+
		"?, ?, ?, ?, "+
		"?, ?, ?, ?, ?, "+
		"?, ?, ?, ?, ?, ?)",
		state.sessionNo, packet.SequenceID, packet.Timestamp,
		string(batteryVoltages), string(batteryTemperatures), inner.SpentMilliAmpHours, inner.SpentMilliWattHours, inner.Current, inner.PercentSOC,
		inner.HydroCurrent, inner.HydroPPM, string(hydroTemperatures),
		inner.TemperatureSMPS, inner.TemperatureEngineDriver, inner.VCEngineDriver[0], inner.VCEngineDriver[1], inner.VCTelemetry[0], inner.VCTelemetry[1], inner.VCSMPS[0], inner.VCSMPS[1], inner.VCBMS[0], inner.VCBMS[1],
		inner.Speed, inner.RPM, inner.VCEngine[0], inner.VCEngine[1],
		inner.Latitude, inner.Longitude, inner.Gyro[0], inner.Gyro[1], inner.Gyro[2],
		inner.QueueFillAmount, inner.TickCounter, inner.FreeHeap, inner.AllocCount, inner.FreeCount, inner.CPUUsage)*/

	if err != nil {
		log.Printf("inserting a new packet failed: %s", err)
	}
}

func (state *State) AdvanceState() uint32 {
	state.receivedPacketCt++

	result := rng.GenericRotLeft(state.currentRNGVector[0]+state.currentRNGVector[3], 7) + state.currentRNGVector[0]

	t := state.currentRNGVector[1] << 9

	state.currentRNGVector[2] ^= state.currentRNGVector[0]
	state.currentRNGVector[3] ^= state.currentRNGVector[1]
	state.currentRNGVector[1] ^= state.currentRNGVector[2]
	state.currentRNGVector[0] ^= state.currentRNGVector[3]

	state.currentRNGVector[2] ^= t

	state.currentRNGVector[3] = rng.GenericRotLeft(state.currentRNGVector[3], 11)

	return result
}

func (state *State) GetDelay() float64 {
	sum := float64(0)

	for _, v := range state.delayAveragingWindow {
		sum += float64(v)
	}

	return sum / float64(len(state.delayAveragingWindow))
}
