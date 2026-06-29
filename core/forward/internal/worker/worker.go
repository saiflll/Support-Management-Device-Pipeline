package worker

import (
	"IoTT/internal/config"
	"IoTT/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// EnvSensorBatchData menampung data gabungan temp + rh untuk satu sensor
type EnvSensorBatchData struct {
	AreaID int
	No     int
	Temp   *float64
	RH     *float64
	TS     string
}

// TempBatchData digunakan oleh processor saat hanya ada data suhu
type TempBatchData struct {
	Value  float64
	AreaID int
	No     int
	TS     string
}

// RhBatchData digunakan oleh processor saat ada data kelembaban
type RhBatchData struct {
	Value  float64
	AreaID int
	No     int
	TS     string
}

// ProxBatchData untuk data proximity/pintu
type ProxBatchData struct {
	Value  int
	DoorID int
	TS     string
}

// BatchInsertEnvSensor melakukan bulk insert ke tabel env_sensor.
// Menggabungkan data temp dan rh menjadi satu baris per sensor per timestamp.
func BatchInsertEnvSensor(tx *sql.Tx, data []EnvSensorBatchData) error {
	if len(data) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(data))
	valueArgs := make([]interface{}, 0, len(data)*5)
	i := 1
	for _, d := range data {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i, i+1, i+2, i+3, i+4))
		valueArgs = append(valueArgs, d.AreaID, d.No, d.Temp, d.RH, d.TS)
		i += 5
	}

	stmt := fmt.Sprintf("INSERT INTO env_sensor (area_id, no, temp, rh, ts) VALUES %s", strings.Join(valueStrings, ","))
	_, err := tx.Exec(stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("error executing bulk insert for env_sensor: %w", err)
	}
	return nil
}

// BatchInsertProx melakukan bulk insert ke tabel prox.
func BatchInsertProx(tx *sql.Tx, data []ProxBatchData) error {
	if len(data) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(data))
	valueArgs := make([]interface{}, 0, len(data)*3)
	i := 1
	for _, d := range data {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", i, i+1, i+2))
		valueArgs = append(valueArgs, d.Value, d.DoorID, d.TS)
		i += 3
	}

	stmt := fmt.Sprintf("INSERT INTO prox (value, door_id, ts) VALUES %s", strings.Join(valueStrings, ","))
	_, err := tx.Exec(stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("error executing bulk insert for prox data: %w", err)
	}
	return nil
}

// SaveFailedBatch menyimpan payload yang gagal dipublish ke MQTT ke tabel failed_batch.
// Dipanggil oleh forwarder saat publish ke cloud gagal.
func SaveFailedBatch(pipelineID int, destTopic, brokerURL string, payload []byte) {
	db := database.GetDB()
	if db == nil {
		log.Printf("⚠️ SaveFailedBatch: DB tidak tersedia, payload pipeline %d hilang.", pipelineID)
		return
	}
	_, err := db.Exec(
		`INSERT INTO failed_batch (pipeline_id, payload_json, dest_topic, broker_url) VALUES ($1, $2, $3, $4)`,
		pipelineID, string(payload), destTopic, brokerURL,
	)
	if err != nil {
		log.Printf("❌ SaveFailedBatch: gagal simpan payload pipeline %d ke DB: %v", pipelineID, err)
	} else {
		log.Printf("💾 SaveFailedBatch: payload pipeline %d tersimpan sebagai fallback.", pipelineID)
	}
}

// FailedBatchRow merepresentasikan satu baris dari tabel failed_batch.
type FailedBatchRow struct {
	ID         int
	PipelineID int
	Payload    []byte
	DestTopic  string
	BrokerURL  string
	RetryCount int
}

// LoadFailedBatches mengambil semua batch yang belum berhasil dikirim (max 50).
func LoadFailedBatches() ([]FailedBatchRow, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("DB tidak tersedia")
	}

	rows, err := db.Query(
		`SELECT id, pipeline_id, payload_json, dest_topic, broker_url, retry_count
		 FROM failed_batch
		 ORDER BY failed_at ASC
		 LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []FailedBatchRow
	for rows.Next() {
		var b FailedBatchRow
		var payloadStr string
		if err := rows.Scan(&b.ID, &b.PipelineID, &payloadStr, &b.DestTopic, &b.BrokerURL, &b.RetryCount); err != nil {
			continue
		}
		b.Payload = []byte(payloadStr)
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// DeleteFailedBatch menghapus satu baris failed_batch setelah retry berhasil.
func DeleteFailedBatch(id int) {
	db := database.GetDB()
	if db == nil {
		return
	}
	db.Exec(`DELETE FROM failed_batch WHERE id = $1`, id)
}

// IncrementRetryCount menambah counter retry untuk batch yang gagal di-retry.
func IncrementRetryCount(id int) {
	db := database.GetDB()
	if db == nil {
		return
	}
	db.Exec(`UPDATE failed_batch SET retry_count = retry_count + 1 WHERE id = $1`, id)
}

// SafetyStatus hasil evaluasi ambang batas sensor
type SafetyStatus struct {
	IsAlert   bool
	Message   string
	Severity  string
	Threshold float64
}

func getThresholdConfig(thresholds []config.ThresholdConfig, areaID int, sensorNo int) (*config.Threshold, bool) {
	for _, t := range thresholds {
		if t.AreaID == areaID && t.SensorNo == sensorNo {
			return &t.Config, true
		}
	}
	return nil, false
}

func getMessageConfig(areaID int, sensorType string) (*config.MessageConfig, bool) {
	for _, m := range config.MessageConfigs {
		if m.AreaID == areaID && m.SensorType == sensorType {
			return &m, true
		}
	}
	return nil, false
}

// EvaluateTemp mengevaluasi nilai suhu terhadap threshold yang dikonfigurasi.
func EvaluateTemp(areaID int, sensorNo int, currentValue float64) SafetyStatus {
	status := SafetyStatus{IsAlert: false, Severity: "NORMAL"}
	sensorCfg, ok := getThresholdConfig(config.TempThresholds, areaID, sensorNo)
	if !ok {
		return status
	}
	msgCfg, _ := getMessageConfig(areaID, "temp")
	location := database.GetAreaName(areaID)
	if sensorCfg.Type == "ambient" {
		return status
	}
	if sensorCfg.UpperCritical != nil && currentValue > sensorCfg.UpperCritical.Limit {
		status.IsAlert = true
		status.Severity = "KRITIS_ATAS"
		status.Threshold = sensorCfg.UpperCritical.Limit
		if msgCfg != nil && msgCfg.UpperCriticalMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.UpperCriticalMsg, config.PbP, location, currentValue, config.KtT, status.Threshold, config.IPbK, location)
		}
		return status
	}
	if sensorCfg.UpperWarning != nil && currentValue > sensorCfg.UpperWarning.Limit {
		status.IsAlert = true
		status.Severity = "WASPADA_ATAS"
		status.Threshold = sensorCfg.UpperWarning.Limit
		if msgCfg != nil && msgCfg.UpperWarningMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.UpperWarningMsg, config.PwP, location, currentValue, config.KmT, status.Threshold, config.IPbW, location)
		}
		return status
	}
	if sensorCfg.LowerCritical != nil && currentValue < sensorCfg.LowerCritical.Limit {
		status.IsAlert = true
		status.Severity = "KRITIS_BAWAH"
		status.Threshold = sensorCfg.LowerCritical.Limit
		if msgCfg != nil && msgCfg.LowerCriticalMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.LowerCriticalMsg, config.PbP, location, currentValue, config.KtR, status.Threshold, config.IPbK, location)
		}
		return status
	}
	if sensorCfg.LowerWarning != nil && currentValue < sensorCfg.LowerWarning.Limit {
		status.IsAlert = true
		status.Severity = "WASPADA_BAWAH"
		status.Threshold = sensorCfg.LowerWarning.Limit
		if msgCfg != nil && msgCfg.LowerWarningMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.LowerWarningMsg, config.PwP, location, currentValue, config.KmR, status.Threshold, config.IPbW, location)
		}
		return status
	}
	return status
}

// EvaluateRh mengevaluasi nilai kelembaban terhadap threshold yang dikonfigurasi.
func EvaluateRh(areaID int, sensorNo int, currentValue float64) SafetyStatus {
	status := SafetyStatus{IsAlert: false, Severity: "NORMAL"}
	sensorCfg, ok := getThresholdConfig(config.RhThresholds, areaID, sensorNo)
	if !ok {
		return status
	}
	msgCfg, _ := getMessageConfig(areaID, "rh")
	location := database.GetAreaName(areaID)
	if sensorCfg.UpperCritical != nil && currentValue > sensorCfg.UpperCritical.Limit {
		status.IsAlert = true
		status.Severity = "KRITIS_ATAS_RH"
		status.Threshold = sensorCfg.UpperCritical.Limit
		if msgCfg != nil && msgCfg.UpperCriticalMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.UpperCriticalMsg, config.PbP, location, currentValue, config.KtT, status.Threshold, config.IPbK, location)
		}
		return status
	}
	if sensorCfg.UpperWarning != nil && currentValue > sensorCfg.UpperWarning.Limit {
		status.IsAlert = true
		status.Severity = "WASPADA_ATAS_RH"
		status.Threshold = sensorCfg.UpperWarning.Limit
		if msgCfg != nil && msgCfg.UpperWarningMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.UpperWarningMsg, config.PwP, location, currentValue, config.KmT, status.Threshold, config.IPbW, location)
		}
		return status
	}
	if sensorCfg.LowerCritical != nil && currentValue < sensorCfg.LowerCritical.Limit {
		status.IsAlert = true
		status.Severity = "KRITIS_BAWAH_RH"
		status.Threshold = sensorCfg.LowerCritical.Limit
		if msgCfg != nil && msgCfg.LowerCriticalMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.LowerCriticalMsg, config.PbP, location, currentValue, config.KtR, status.Threshold, config.IPbK, location)
		}
		return status
	}
	if sensorCfg.LowerWarning != nil && currentValue < sensorCfg.LowerWarning.Limit {
		status.IsAlert = true
		status.Severity = "WASPADA_BAWAH_RH"
		status.Threshold = sensorCfg.LowerWarning.Limit
		if msgCfg != nil && msgCfg.LowerWarningMsg != "" {
			status.Message = fmt.Sprintf(msgCfg.LowerWarningMsg, config.PwP, location, currentValue, config.KmR, status.Threshold, config.IPbW, location)
		}
		return status
	}
	return status
}

// marshalPayload helper untuk serialize payload ke JSON bytes (dipakai oleh forwarder)
func MarshalPayload(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Referensi timestamp untuk penggunaan internal
var _ = time.Now