package mdcw

import "time"

type Payload struct {
	Ts         interface{} `json:"ts"`
	Reg2       int         `json:"reg2"`
	Reg5       int         `json:"reg5"`
	Reg114     int         `json:"reg114"`
	Total      int         `json:"total"`
	Code       int         `json:"code"`
	Weight     int         `json:"weight"`
	Prefix     string      `json:"prefix"`
	NodePrefix string      `json:"node_prefix"`
	Data       struct {
		Reg2   int `json:"reg2"`
		Reg5   int `json:"reg5"`
		Reg114 int `json:"reg114"`
	} `json:"data"`
}

type Record struct {
	ID              int       `json:"id"`
	Ts              string    `json:"ts"`
	Reg2            int       `json:"reg2"`
	Reg5            int       `json:"reg5"`
	Reg114          int       `json:"reg114"`
	WeightFormatted string    `json:"weight_formatted"`
	Prefix          string    `json:"prefix"`
	DataType        string    `json:"data_type"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"created_at"`
}

type Summary struct {
	Prefix     string `json:"prefix"`
	TotalCount int    `json:"total_count"`
	OkCount    int    `json:"ok_count"`
	UnderCount int    `json:"under_count"`
	OverCount  int    `json:"over_count"`
	MetalCount int    `json:"metal_count"`
	AvgWeight  int    `json:"avg_weight"`
	MinWeight  int    `json:"min_weight"`
	MaxWeight  int    `json:"max_weight"`
	OkWeight   int    `json:"ok_weight"`
	SumWeight  int    `json:"sum_weight"`
}

type ConveyorRecord struct {
	IDRecord          int64     `json:"id_record"`
	Factory           string    `json:"factory"`
	KodeProduk        string    `json:"kode_produk"`
	QtyPerPack        int       `json:"qty_per_pack"`
	PackPerKarton     int       `json:"pack_per_karton"`
	GramasiPack       float64   `json:"gramasi_pack"`
	GramasiKarton     int       `json:"gramasi_karton"`
	TanggalProduksi   string    `json:"tanggal_produksi"`
	Shift             int       `json:"shift"`
	TanggalBestBefore string    `json:"tanggal_best_before"`
	KodeKetentuan     string    `json:"kode_ketentuan"`
	KodeBatch         string    `json:"kode_batch"`
	TanggalRecord     time.Time `json:"tanggal_record"`
}

