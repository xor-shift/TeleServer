package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"github.com/xor-shift/teleserver/ingest"
	"log"
	"os"
)

const (
	bigInsertQuery = "" +
		"INSERT INTO packets (session_id, packet_order, reported_time" +
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

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading dotenv failed: %s", err)
	}
}

func main() {
	var err error

	var amqpConn *amqp.Connection
	var amqpChan *amqp.Channel
	var amqpQueue amqp.Queue
	var amqpConsumer <-chan amqp.Delivery
	var db *sql.DB

	if amqpConn, err = amqp.Dial(os.Getenv("AMQP_URL")); err != nil {
		log.Fatalf("Failed to dial amqp: %s", err)
	}

	if amqpChan, err = amqpConn.Channel(); err != nil {
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
	}

	if amqpQueue, err = amqpChan.QueueDeclare(
		"full_packet_queue_db", // name
		false,                  // durable
		false,                  // delete when unused
		true,                   // exclusive
		false,                  // no-wait
		nil,                    // arguments
	); err != nil {
		log.Fatalf("Failed to declare an amqp queue: %s", err)
	}

	if err = amqpChan.QueueBind(
		amqpQueue.Name, // queue name
		"",             // routing key
		"full_packets", // exchange
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to bind an amqp queue: %s", err)
	}

	amqpConsumer, err = amqpChan.Consume(
		amqpQueue.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)

	dbConfig := mysql.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASSWORD"),
		Addr:                 os.Getenv("DB_ADDRESS"),
		DBName:               os.Getenv("DB_NAME"),
		Collation:            "utf8mb4_general_ci",
		Net:                  "tcp",
		AllowNativePasswords: true,
	}

	if db, err = sql.Open("mysql", dbConfig.FormatDSN()); err != nil {
		log.Fatalln(err)
	}

	(func(any) {})(db)

	processPacket := func(amqpPacket ingest.AMQPPacket) error {
		tx, err := db.BeginTx(context.TODO(), nil)
		if err != nil {
			//log.Printf("failed writing a message to the db (BeginTx): %s", err)
			return err
		}
		defer tx.Rollback()

		var stmt *sql.Stmt
		if stmt, err = tx.Prepare(bigInsertQuery); err != nil {
			return err
		}
		defer stmt.Close()

		packet := amqpPacket.Packet
		inner := packet.Inner.(ingest.FullPacket)

		batteryVoltages, _ := json.Marshal(inner.BatteryVoltages[:])
		batteryTemperatures, _ := json.Marshal(inner.BatteryTemperatures[:])
		hydroTemperatures, _ := json.Marshal(inner.HydroTemperatures[:])

		if _, err = stmt.Exec(
			amqpPacket.SessionID, packet.SequenceID, packet.Timestamp,
			string(batteryVoltages), string(batteryTemperatures), inner.SpentMilliAmpHours, inner.SpentMilliWattHours, inner.Current, inner.PercentSOC,
			inner.HydroCurrent, inner.HydroPPM, string(hydroTemperatures),
			inner.TemperatureSMPS, inner.TemperatureEngineDriver, inner.VCEngineDriver[0], inner.VCEngineDriver[1], inner.VCTelemetry[0], inner.VCTelemetry[1], inner.VCSMPS[0], inner.VCSMPS[1], inner.VCBMS[0], inner.VCBMS[1],
			inner.Speed, inner.RPM, inner.VCEngine[0], inner.VCEngine[1],
			inner.Latitude, inner.Longitude, inner.Gyro[0], inner.Gyro[1], inner.Gyro[2],
			inner.QueueFillAmount, inner.TickCounter, inner.FreeHeap, inner.AllocCount, inner.FreeCount, inner.CPUUsage,
		); err != nil {
			return err
		}

		return tx.Commit()
	}

	for delivery := range amqpConsumer {
		buffer := bytes.NewBuffer(delivery.Body)
		decoder := gob.NewDecoder(buffer)
		var packet ingest.AMQPPacket
		if err := decoder.Decode(&packet); err != nil {
			log.Printf("error decoding a packet with gob: %s", err)
		}

		processPacket(packet)
	}
}
