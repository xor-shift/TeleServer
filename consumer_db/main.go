package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"github.com/xor-shift/teleserver/common"
	"log"
	"os"
)

const (
	bigInsertQuery = "" +
		"INSERT INTO packets (session_id, packet_order, reported_time" +
		", battery_voltages, battery_temperatures, spent_mah, spent_mwh, curr, percent_soc" +
		", hydro_curr, hydro_ppm, hydro_temp" +
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

	var consumer *common.AMQPConsumer
	var db *sql.DB

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

	if consumer, err = common.NewAMQPConsumer(
		"consumer_db_queue",
		"consumer_db_consumer",
		func(delivery amqp.Delivery) error {
			var amqpPacket common.AMQPPacket

			if amqpPacket, err = common.ParseAMQPPacket(&delivery); err != nil {
				log.Printf("error decoding a packet with gob: %s", err)
			}

			var tx *sql.Tx
			tx, err = db.BeginTx(context.TODO(), nil)
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
			inner := packet.Inner.(common.FullPacket)

			batteryVoltages, _ := json.Marshal(inner.BatteryVoltages[:])
			batteryTemperatures, _ := json.Marshal(inner.BatteryTemperatures[:])

			if _, err = stmt.Exec(
				amqpPacket.SessionID, packet.SequenceID, packet.Timestamp,
				string(batteryVoltages), string(batteryTemperatures), inner.SpentMilliAmpHours, inner.SpentMilliWattHours, inner.Current, inner.PercentSOC,
				inner.HydroCurrent, inner.HydroPPM, inner.HydroTemperature,
				inner.TemperatureSMPS, inner.TemperatureEngineDriver, inner.VCEngineDriver[0], inner.VCEngineDriver[1], inner.VCTelemetry[0], inner.VCTelemetry[1], inner.VCSMPS[0], inner.VCSMPS[1], inner.VCBMS[0], inner.VCBMS[1],
				inner.Speed, inner.RPM, inner.VCEngine[0], inner.VCEngine[1],
				inner.Latitude, inner.Longitude, inner.Gyro[0], inner.Gyro[1], inner.Gyro[2],
				inner.QueueFillAmount, inner.TickCounter, inner.FreeHeap, inner.AllocCount, inner.FreeCount, inner.CPUUsage,
			); err != nil {
				return err
			}

			return tx.Commit()
		}); err != nil {
		log.Fatalln(err)
	}

	if err = consumer.Start(); err != nil {
		log.Fatalln(err)
	}

	consumer.Wait()
}
