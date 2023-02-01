CREATE TABLE packets (
    session_id    int(11)   NOT NULL DEFAULT 0,
    packet_order  int(11)   NOT NULL DEFAULT 0,
    insert_time   timestamp NOT NULL DEFAULT current_timestamp(),
    reported_time timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',

    battery_voltages     text  NOT NULL DEFAULT '[]',
    battery_temperatures text  NOT NULL DEFAULT '[]',
    spent_mah            float NOT NULL DEFAULT 0,
    spent_mwh            float NOT NULL DEFAULT 0,
    curr                 float NOT NULL DEFAULT 0,
    percent_soc          float NOT NULL DEFAULT 0,

    hydro_curr  float NOT NULL DEFAULT 0,
    hydro_ppm   float NOT NULL DEFAULT 0,
    hydro_temps text  NOT NULL DEFAULT '[]',

    temperature_smps          float NOT NULL DEFAULT 0,
    temperature_engine_driver float NOT NULL DEFAULT 0,
    voltage_engine_driver     float NOT NULL DEFAULT 0,
    current_engine_driver     float NOT NULL DEFAULT 0,
    voltage_telemetry         float NOT NULL DEFAULT 0,
    current_telemetry         float NOT NULL DEFAULT 0,
    voltage_smps              float NOT NULL DEFAULT 0,
    current_smps              float NOT NULL DEFAULT 0,
    voltage_bms               float NOT NULL DEFAULT 0,
    current_bms               float NOT NULL DEFAULT 0,

    speed          float NOT NULL DEFAULT 0,
    rpm            float NOT NULL DEFAULT 0,
    voltage_engine float NOT NULL DEFAULT 0,
    current_engine float NOT NULL DEFAULT 0,

    latitude  float NOT NULL DEFAULT 0,
    longitude float NOT NULL DEFAULT 0,
    gyro_x    float NOT NULL DEFAULT 0,
    gyro_y    float NOT NULL DEFAULT 0,
    gyro_z    float NOT NULL DEFAULT 0,

    queue_fill_amt int UNSIGNED NOT NULL DEFAULT 0,
    tick_counter   int UNSIGNED NOT NULL DEFAULT 0,
    free_heap      int UNSIGNED NOT NULL DEFAULT 0,
    alloc_count    int UNSIGNED NOT NULL DEFAULT 0,
    free_count     int UNSIGNED NOT NULL DEFAULT 0,
    cpu_usage      float        NOT NULL DEFAULT 0,

    PRIMARY KEY (`session_id`,`packet_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE packets_json (
    session_id    int(11)   NOT NULL DEFAULT 0,
    packet_order  int(11)   NOT NULL DEFAULT 0,
    insert_time   timestamp NOT NULL DEFAULT current_timestamp(),
    reported_time timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',

    inner_data longtext NOT NULL DEFAULT '{}'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
