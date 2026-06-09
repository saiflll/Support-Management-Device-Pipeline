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

func InitGoogleSheets() error {
	crdB64 := os.Getenv("GOOGLE_SHEETS_CREDENTIALS")
	if crdB64 == "" {
		return fmt.Errorf("GOOGLE_SHEETS_CREDENTIALS not set")
	}
	crdJson, err := base64.StdEncoding.DecodeString(crdB64)
	if err != nil {
		return fmt.Errorf("failed to decode credentials: %w", err)
	}
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(crdJson))
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
		sheetName = "Production Data"
	}

	log.Println("✅ Google Sheets initialized successfully")
	return nil
}

func CreateSheetIfNotExists() {
	if sheetsService == nil {
		return
	}
	rngBc := fmt.Sprintf("%s!A1:F1", sheetName)
	res, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, rngBc).Do()

	if err != nil || len(res.Values) == 0 {
		hdr := []interface{}{"Timestamp", "Pack Count", "Status", "Weight (g)", "Prefix", "Created At"}
		rngVal := &sheets.ValueRange{
			Values: [][]interface{}{hdr},
		}

		rngHdr := fmt.Sprintf("%s!A1", sheetName)
		_, err := sheetsService.Spreadsheets.Values.Update(
			spreadsheetID,
			rngHdr,
			rngVal,
		).ValueInputOption("RAW").Do()

		if err != nil {
			log.Printf("Warning: Failed to create header: %v", err)
		} else {
			log.Printf("✅ Created header in sheet '%s'", sheetName)
		}
	}
}

type Payload struct {
	Ts     string `json:"ts"`
	Reg2   int    `json:"reg2"`
	Reg5   int    `json:"reg5"`
	Reg114 int    `json:"reg114"`
	Prefix string `json:"prefix"`
}

func AppendToSheet(psn Payload) error {
	if sheetsService == nil {
		return fmt.Errorf("sheets service not initialized")
	}

	sts := "Unknown"
	switch psn.Reg5 {
	case 1:
		sts = "OK"
	case 2:
		sts = "Under"
	case 3:
		sts = "Over"
	}

	brs := []interface{}{
		psn.Ts,
		psn.Reg2,
		sts,
		psn.Reg114,
		psn.Prefix,
		time.Now().Format("2006-01-02 15:04:05"),
	}

	rngVal := &sheets.ValueRange{
		Values: [][]interface{}{brs},
	}

	rngApp := fmt.Sprintf("%s!A:F", sheetName)
	_, err := sheetsService.Spreadsheets.Values.Append(
		spreadsheetID,
		rngApp,
		rngVal,
	).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()

	if err != nil {
		return fmt.Errorf("failed to append to sheet: %w", err)
	}

	log.Printf("📊 Exported to Sheets: %s | %s | %dg", psn.Prefix, sts, psn.Reg114)
	return nil
}

func PayloadFromJSON(dt []byte) (Payload, error) {
	var p Payload
	err := json.Unmarshal(dt, &p)
	return p, err
}
