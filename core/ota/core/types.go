package core

import (
	"encoding/json"
	"regexp"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type FlexString string

func (fs *FlexString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*fs = FlexString(s)
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err == nil {
		*fs = FlexString(strconv.FormatFloat(f, 'f', -1, 64))
		return nil
	}
	var bl bool
	if err := json.Unmarshal(b, &bl); err == nil {
		*fs = FlexString(strconv.FormatBool(bl))
		return nil
	}
	*fs = FlexString(string(b))
	return nil
}

func (fs FlexString) String() string {
	return string(fs)
}

type NodeInfo struct {
	Status       string                 `json:"status,omitempty"`
	RamFreeBytes int64                  `json:"ram_free_bytes,omitempty"`
	SD_OK        *bool                  `json:"sd_ok,omitempty"`
	Ck           FlexString             `json:"ck,omitempty"`
	Area         FlexString             `json:"area,omitempty"`
	No           FlexString             `json:"no,omitempty"`
	MinT1        float64                `json:"min_t1,omitempty"`
	MaxT1        float64                `json:"max_t1,omitempty"`
	MinT2        float64                `json:"min_t2,omitempty"`
	MaxT2        float64                `json:"max_t2,omitempty"`
	MinT3        float64                `json:"min_t3,omitempty"`
	MaxT3        float64                `json:"max_t3,omitempty"`
	SHTSuhuMin   float64                `json:"sht_suhu_min,omitempty"`
	SHTSuhuMax   float64                `json:"sht_suhu_max,omitempty"`
	SHTHumMin    float64                `json:"sht_humidity_min,omitempty"`
	SHTHumMax    float64                `json:"sht_humidity_max,omitempty"`
	DoorDelay    uint32                 `json:"door_logic_delay,omitempty"`
	Reboot       int                    `json:"reboot,omitempty"`
	Interval     uint64                 `json:"interval,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	Updated      string                 `json:"updated,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Prefix       string                 `json:"node_prefix,omitempty"`
	Version      string                 `json:"version,omitempty"`
	AppMode      string                 `json:"app_mode,omitempty"`
	Trans        string                 `json:"trans,omitempty"`
	PassCode     string                 `json:"pass_code,omitempty"`
	Relay        bool                   `json:"relay"`
	NoT1         int                    `json:"no_t1"`
	NoT2         int                    `json:"no_t2"`
	NoT3         int                    `json:"no_t3"`
	NoSHT        int                    `json:"no_sht"`
	NoP1         int                    `json:"no_p1"`
	NoP2         int                    `json:"no_p2"`
	NoP3         int                    `json:"no_p3"`
	CurT1        float64                `json:"cur_t1"`
	CurT2        float64                `json:"cur_t2"`
	CurT3        float64                `json:"cur_t3"`
	CurP1        int                    `json:"cur_p1"`
	CurP2        int                    `json:"cur_p2"`
	CurP3        int                    `json:"cur_p3"`
	CurSHT_T     float64                `json:"cur_sht_t"`
	CurSHT_H     float64                `json:"cur_sht_h"`
	Logs         []string               `json:"logs,omitempty"`
	FullConfig   map[string]interface{} `json:"full_config,omitempty"`
}

type FileInfo struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	UploadTime time.Time `json:"upload_time"`
	Size       int64     `json:"size"`
}

type ModelConfig struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Fields      []Field `json:"fields"`
	Command     string  `json:"command"`
}

type Field struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Step        *float64 `json:"step,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type EMQXWebhook struct {
	Event    string `json:"event"`
	ClientID string `json:"clientid"`
	Reason   string `json:"reason,omitempty"`
}

var (
	MqttClient mqtt.Client
	NodeMutex  sync.RWMutex
	NodeStatus = make(map[string]*NodeInfo)
	FileMutex  sync.RWMutex
	FileInfos  = make(map[string]FileInfo)

	Store            *session.Store
	TelegramBotToken string
	TelegramChatID   string
	WebhookToken     string

	LastAlarmState = make(map[string]bool)
	AlarmMutex     sync.Mutex

	MacRegex = regexp.MustCompile(`[0-9a-fA-F]{12}`)
)

var commonTempFields = []Field{
	{Name: "ck", Label: "Central Kitchen", Type: "number", Required: true},
	{Name: "area", Label: "Area ID", Type: "number", Required: true},
	{Name: "no", Label: "Box ID", Type: "number", Required: true},
	{Name: "no_t1", Label: "No T1", Type: "number", Required: false},
	{Name: "no_t2", Label: "No T2", Type: "number", Required: false},
	{Name: "no_t3", Label: "No T3", Type: "number", Required: false},
	{Name: "no_sht", Label: "No SHT", Type: "number", Required: false},
	{Name: "no_p1", Label: "No P1", Type: "number", Required: false},
	{Name: "no_p2", Label: "No P2", Type: "number", Required: false},
	{Name: "no_p3", Label: "No P3", Type: "number", Required: false},
	{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "TEMP"},
	{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
	{Name: "door_logic_delay", Label: "Door Alarm Delay (ms)", Type: "number", Required: false},
}

var commonMdcwFields = []Field{
	{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "MDCW"},
	{Name: "interval", Label: "Interval (ms)", Type: "number", Required: true},
}

var ModelRegistry = map[string]ModelConfig{
	"TEMP": {
		Name: "TEMP", DisplayName: "Temperature Sensor (Universal)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t3", Label: "Min T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t3", Label: "Max T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|1": {
		Name: "M1", DisplayName: "TEMP-M1 (1 DS)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|2": {
		Name: "M2", DisplayName: "TEMP-M2 (Modbus + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|3": {
		Name: "M3", DisplayName: "TEMP-M3 (2 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|4": {
		Name: "M4", DisplayName: "TEMP-M4 (3 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t3", Label: "Min T3", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t3", Label: "Max T3", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|5": {
		Name: "M5", DisplayName: "TEMP-M5 (1 DS + 2 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|6": {
		Name: "M6", DisplayName: "TEMP-M6 (1 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|7": {
		Name: "M7", DisplayName: "TEMP-M7 (1 Prox)", Command: "set_config",
		Fields: commonTempFields,
	},
	"TEMP|8": {
		Name: "M8", DisplayName: "TEMP-M8 (Modbus Only)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "sht_suhu_min", Label: "Min SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_suhu_max", Label: "Max SHT Temp", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "sht_humidity_min", Label: "Min SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
			{Name: "sht_humidity_max", Label: "Max SHT Hum", Type: "number", Required: false, Step: floatPtr(1)},
		}...),
	},
	"TEMP|9": {
		Name: "M9", DisplayName: "TEMP-M9 (1 DS Only)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|10": {
		Name: "M10", DisplayName: "TEMP-M10 (2 DS + 1 Prox)", Command: "set_config",
		Fields: append(commonTempFields, []Field{
			{Name: "min_t1", Label: "Min T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t1", Label: "Max T1", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "min_t2", Label: "Min T2", Type: "number", Required: false, Step: floatPtr(0.1)},
			{Name: "max_t2", Label: "Max T2", Type: "number", Required: false, Step: floatPtr(0.1)},
		}...),
	},
	"TEMP|11": {
		Name: "M11", DisplayName: "TEMP-M11 (2 Prox)", Command: "set_config",
		Fields: commonTempFields,
	},
	"MDCW": {
		Name:        "MDCW",
		DisplayName: "MDCW Weighing",
		Command:     "set_config",
		Fields:      commonMdcwFields,
	},
	"V1": {
		Name:        "V1",
		DisplayName: "MDCW V1 (Standard)",
		Command:     "set_config",
		Fields:      commonMdcwFields,
	},
	"V2": {
		Name:        "V2",
		DisplayName: "MDCW V2 (Prox)",
		Command:     "set_config",
		Fields: append(commonMdcwFields, []Field{
			{Name: "prox_nc0", Label: "Prox 1 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc1", Label: "Prox 2 (0:NO, 1:NC)", Type: "number", Required: false},
			{Name: "prox_nc2", Label: "Prox 3 (0:NO, 1:NC)", Type: "number", Required: false},
		}...),
	},
	"TROLI": {
		Name:        "TROLI",
		DisplayName: "Troli Scanner (Mode A/B)",
		Command:     "set_config",
		Fields: []Field{
			{Name: "app_mode", Label: "App Mode", Type: "select", Required: true, Options: []string{"A", "B"}},
			{Name: "trans", Label: "Trans (IN/OUT) - Mode A Only", Type: "select", Required: false, Options: []string{"IN", "OUT"}},
			{Name: "pass_code", Label: "Target Product Code - Mode B", Type: "text", Required: false, Placeholder: "e.g., 100209"},
		},
	},
	"GENERIC": {
		Name: "GENERIC", DisplayName: "Unregistered Device", Command: "set_config",
		Fields: []Field{
			{Name: "node_prefix", Label: "Node Prefix", Type: "text", Required: false, Placeholder: "NODE-"},
		},
	},
}

func floatPtr(f float64) *float64 {
	return &f
}
