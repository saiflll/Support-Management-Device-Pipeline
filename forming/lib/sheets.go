package lib

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	sheetsService *sheets.Service
	spreadsheetID string
	sheetName     string
)

// InitGoogleSheets initializes Google Sheets API client
func InitGoogleSheets() error {
	// Read Base64-encoded credentials from environment
	credsBase64 := os.Getenv("GOOGLE_SHEETS_CREDENTIALS")
	if credsBase64 == "" {
		return fmt.Errorf("GOOGLE_SHEETS_CREDENTIALS not set")
	}

	// Decode Base64
	credsJSON, err := base64.StdEncoding.DecodeString(credsBase64)
	if err != nil {
		return fmt.Errorf("failed to decode credentials: %w", err)
	}

	// Create Sheets service
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(credsJSON))
	if err != nil {
		return fmt.Errorf("failed to create sheets service: %w", err)
	}

	sheetsService = srv
	spreadsheetID = os.Getenv("GOOGLE_SPREADSHEET_ID")
	sheetName = os.Getenv("GOOGLE_SHEET_NAME")

	if spreadsheetID == "" {
		return fmt.Errorf("GOOGLE_SPREADSHEET_ID not set")
	}
	if sheetName == "" {
		sheetName = "Production Data" // Default sheet name
	}

	log.Println("✅ Google Sheets initialized successfully")
	return nil
}

// CreateSheetIfNotExists creates the sheet and header row if not exists
func CreateSheetIfNotExists() {
	if sheetsService == nil {
		return
	}

	// Check if sheet exists, if not create header
	readRange := fmt.Sprintf("%s!A1:F1", sheetName)
	resp, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()

	if err != nil || len(resp.Values) == 0 {
		// Sheet might not exist or header is missing, add header
		header := []interface{}{"Timestamp", "Pack Count", "Status", "Weight (g)", "Prefix", "Created At"}
		valueRange := &sheets.ValueRange{
			Values: [][]interface{}{header},
		}

		headerRange := fmt.Sprintf("%s!A1", sheetName)
		_, err := sheetsService.Spreadsheets.Values.Update(
			spreadsheetID,
			headerRange,
			valueRange,
		).ValueInputOption("RAW").Do()

		if err != nil {
			log.Printf("Warning: Failed to create header: %v", err)
		} else {
			log.Printf("✅ Created header in sheet '%s'", sheetName)
		}
	}
}

// Payload structure untuk export
type Payload struct {
	Ts     string `json:"ts"`
	Reg2   int    `json:"reg2"`
	Reg5   int    `json:"reg5"`
	Reg114 int    `json:"reg114"`
	Prefix string `json:"prefix"`
}

// AppendToSheet appends a row to Google Sheets
func AppendToSheet(payload Payload) error {
	if sheetsService == nil {
		return fmt.Errorf("sheets service not initialized")
	}

	// Map status code to text
	statusText := "Unknown"
	switch payload.Reg5 {
	case 1:
		statusText = "OK"
	case 2:
		statusText = "Under"
	case 3:
		statusText = "Over"
	}

	// Prepare row data
	row := []interface{}{
		payload.Ts,
		payload.Reg2,
		statusText,
		payload.Reg114,
		payload.Prefix,
		time.Now().Format("2006-01-02 15:04:05"),
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	appendRange := fmt.Sprintf("%s!A:F", sheetName)
	_, err := sheetsService.Spreadsheets.Values.Append(
		spreadsheetID,
		appendRange,
		valueRange,
	).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()

	if err != nil {
		return fmt.Errorf("failed to append to sheet: %w", err)
	}

	log.Printf("📊 Exported to Sheets: %s | %s | %dg", payload.Prefix, statusText, payload.Reg114)
	return nil
}

// Helper to convert Payload from JSON
func PayloadFromJSON(data []byte) (Payload, error) {
	var p Payload
	err := json.Unmarshal(data, &p)
	return p, err
}
