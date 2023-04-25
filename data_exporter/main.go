package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"log"
	"os"
	"text/template"
	"time"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading dotenv failed: %s", err)
	}
}

func main() {
	var err error

	var db *sql.DB

	args := struct {
		Session            int    `name:"session" short:"s" help:"session number to export" required:""`
		Out                string `name:"out" short:"o" default:"session_{{.SessionNo}}.csv" help:"File to output to (templated)"`
		Mode               string `name:"mode" short:"m" enum:"electro,hydro" default:"electro" help:"Data mode"`
		Format             string `name:"format" short:"f" enum:"csv,json" default:"csv" help:"Data format"`
		ExportColumnTitles bool   `name:"export_column_titles" negatable:"" default:"true" help:"(applicable only to CSV outputs) whether to include column titles for CSV exports"`
	}{}

	_ = kong.Parse(&args)

	dbConfig := mysql.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASSWORD"),
		Addr:                 os.Getenv("DB_ADDRESS"),
		DBName:               os.Getenv("DB_NAME"),
		Collation:            "utf8mb4_general_ci",
		Net:                  "tcp",
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	if db, err = sql.Open("mysql", dbConfig.FormatDSN()); err != nil {
		log.Fatalln(err)
	}

	defer db.Close()

	type Row struct {
		PacketOrder   int
		TickCounterLF int
		InsertTime    time.Time
		ReportedTime  time.Time

		BatteryVoltages     []float32
		BatteryTemperatures [5]float32
		SpentMAH            float32
		SpentMWH            float32
		Current             float32
		SoC                 float32
		Speed               float32
		RPM                 float32
		Latitude            float32
		Longitude           float32
		Gyro                [3]float32

		HydroCurrent     float32
		HydroPPM         float32
		HydroTemperature float32

		QueueFillAmount int
		HeapFreeAmount  int
		HeapAllocCount  int
		HeapFreeCount   int
		CPUUsage        float32
	}

	const selectQuery string = "SELECT packet_order, tick_counter, insert_time, reported_time, battery_voltages, battery_temperatures, spent_mah, spent_mwh, curr, percent_soc, speed, rpm, latitude, longitude, gyro_x, gyro_y, gyro_z, hydro_curr, hydro_ppm, hydro_temp, queue_fill_amt, free_heap, alloc_count, free_count, cpu_usage FROM packets WHERE session_id=?"

	var sqlRows *sql.Rows
	if sqlRows, err = db.Query(selectQuery, args.Session); err != nil {
		log.Fatalf("Failed to fetch rows for session %d: %s", args.Session, err)
	}

	var rows []Row
	for i := 0; sqlRows.Next(); i++ {
		var row Row

		var voltagesString string
		var temperaturesString string

		if err = sqlRows.Scan(
			&row.PacketOrder, &row.TickCounterLF, &row.InsertTime, &row.ReportedTime,
			&voltagesString, &temperaturesString,
			&row.SpentMAH, &row.SpentMWH, &row.Current, &row.SoC,
			&row.Speed, &row.RPM,
			&row.Latitude, &row.Longitude,
			&row.Gyro[0],
			&row.Gyro[1],
			&row.Gyro[2],
			&row.HydroCurrent,
			&row.HydroPPM,
			&row.HydroTemperature,
			&row.QueueFillAmount,
			&row.HeapFreeAmount,
			&row.HeapAllocCount,
			&row.HeapFreeCount,
			&row.CPUUsage,
		); err != nil {
			log.Fatalf("error while reading row %d of session %d: %s", i, args.Session, err)
		}

		if err = json.Unmarshal([]byte(voltagesString), &row.BatteryVoltages); err != nil {
			log.Fatalf("error while parsing battery voltages of row %d of session %d: %s", i, args.Session, err)
		}

		if err = json.Unmarshal([]byte(temperaturesString), &row.BatteryTemperatures); err != nil {
			log.Fatalf("error while parsing battery voltages of row %d of session %d: %s", i, args.Session, err)
		}

		if args.Mode == "hydro" {
			row.BatteryVoltages = row.BatteryVoltages[0:20]
		}

		rows = append(rows, row)
	}

	db.Close()

	var outFileNameTemplate *template.Template
	if outFileNameTemplate, err = template.New("").Parse(args.Out); err != nil {
		log.Fatalf("error while creating the output filename template: %s", err)
	}

	outFileNameBuf := bytes.Buffer{}

	templateArguments := struct {
		SessionNo int
	}{
		SessionNo: args.Session,
	}

	if err = outFileNameTemplate.Execute(&outFileNameBuf, templateArguments); err != nil {
		log.Fatalf("error while executing the output filename template: %s", err)
	}

	outFileName := outFileNameBuf.String()

	var outFile *os.File
	if outFile, err = os.Create(outFileName); err != nil {
		log.Fatalf("error while creating the output file \"%s\": %s", outFileName, err)
	}

	csvWriter := csv.NewWriter(outFile)

	if args.ExportColumnTitles {
		columns := []string{
			"Packet Order",
			"Seconds Since Boot",
			"Insert Time",
			"Reported Time",
		}

		var cellCount int

		if args.Mode == "hydro" {
			cellCount = 20
		} else {
			cellCount = 27
		}

		for i := 0; i < cellCount; i++ {
			columns = append(columns, fmt.Sprintf("Cell %d", i))
		}

		for i := 0; i < 5; i++ {
			columns = append(columns, fmt.Sprintf("Temp %d", i))
		}

		columns = append(columns, []string{
			"Spent mAh",
			"Spent mWh",
			"Current",
			"SoC",
			"Speed",
			"RPM",
		}...)

		if args.Mode == "hydro" {
			columns = append(columns, []string{
				"Hydrogen Current",
				"Hydrogen PPM",
				"Hydrogen Temperature",
			}...)
		}

		columns = append(columns, []string{
			"Queue Fill Amt",
			"Free Heap Bytes",
			"malloc Calls",
			"free Calls",
		}...)

		_ = csvWriter.Write(columns)
	}

	for _, row := range rows {
		rowStrings := []string{}

		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.PacketOrder))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", float32(row.TickCounterLF)/1000.))
		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.InsertTime.Unix()))
		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.ReportedTime.Unix()))

		for _, v := range row.BatteryVoltages {
			rowStrings = append(rowStrings, fmt.Sprintf("%f", v))
		}

		for _, v := range row.BatteryTemperatures {
			rowStrings = append(rowStrings, fmt.Sprintf("%f", v))
		}

		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.SpentMAH))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.SpentMWH))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.Current))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.SoC))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.Speed))
		rowStrings = append(rowStrings, fmt.Sprintf("%f", row.RPM))

		if args.Mode == "hydro" {
			rowStrings = append(rowStrings, fmt.Sprintf("%f", row.HydroCurrent))
			rowStrings = append(rowStrings, fmt.Sprintf("%f", row.HydroPPM))
			rowStrings = append(rowStrings, fmt.Sprintf("%f", row.HydroTemperature))
		}

		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.QueueFillAmount))
		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.HeapFreeAmount))
		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.HeapAllocCount))
		rowStrings = append(rowStrings, fmt.Sprintf("%d", row.HeapFreeCount))

		csvWriter.Write(rowStrings)
	}

	csvWriter.Flush()
}
