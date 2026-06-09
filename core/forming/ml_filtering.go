package main

import (
	"math"
	"sync"
	"time"
)

// === KONSTANTA TIPE DATA ===

const (
	DataTypeValid = "VALID"
	DataTypeIsen  = "ISEN"
	DataTypeSpam  = "SPAM"
	DataTypeTest  = "TEST"
)

type MachineStats struct {
	LastTs         time.Time
	WeightSum      float64
	WeightCount    int
	AvgWeight      float64
	M2             float64 // For rolling variance (Welford's algorithm)
	LastDelays     []float64
	HistoryLength  int
}

var (
	machineStates   = make(map[string]*MachineStats)
	machineStatesMu sync.Mutex
)

// === ANALISIS DATA ===

// AnalyzeRecord performs lightweight ML filtering on incoming production data
func AnalyzeRecord(pfx string, wgt int) (string, float64) {
	machineStatesMu.Lock()
	defer machineStatesMu.Unlock()

	sts, ada := machineStates[pfx]
	wkt := time.Now()

	if !ada {
		// inisialisasi state untuk mesin baru
		machineStates[pfx] = &MachineStats{
			LastTs:        wkt,
			AvgWeight:     float64(wgt),
			WeightSum:     float64(wgt),
			WeightCount:   1,
			HistoryLength: 50, // Keep rolling stats for 50 records
		}
		return DataTypeValid, 0.7 // New machine, assume valid but medium confidence
	}

	// 1. hitung delay (spam check)
	dly := wkt.Sub(sts.LastTs).Seconds()
	sts.LastTs = wkt

	// 2. weight anomaly (isen/test check)
	// Kami menggunakan toleransi 60-80% (0.4 hingga 1.6 factor) yang disebutkan user
	lwr := sts.AvgWeight * 0.4
	upr := sts.AvgWeight * 1.6

	typ := DataTypeValid
	cfd := 1.0

	// SPAM CHECK: Very high frequency
	if dly < 0.3 {
		typ = DataTypeSpam
		cfd = 0.9
	} else if wgt == 0 {
		typ = DataTypeIsen
		cfd = 1.0
	} else if float64(wgt) < lwr || float64(wgt) > upr {
		// diluar batas toleransi
		typ = DataTypeIsen
		cfd = 0.8
		
		// jika outlier sangat spesifik (misal tepat setengah rata-rata), mungkin TEST
		if wgt > 0 && sts.AvgWeight > 0 {
			rt := float64(wgt) / sts.AvgWeight
			if math.Abs(rt-0.5) < 0.05 || math.Abs(rt-2.0) < 0.05 {
				typ = DataTypeTest
				cfd = 0.7
			}
		}
	}

	// 3. deteksi test pattern
	// Cek jika frekuensi data "terlalu konsisten" (testing manual)
	sts.LastDelays = append(sts.LastDelays, dly)
	if len(sts.LastDelays) > 5 {
		sts.LastDelays = sts.LastDelays[1:]
		
		// varians delay
		var sm, smSq float64
		for _, d := range sts.LastDelays {
			sm += d
			smSq += d * d
		}
		avgDly := sm / float64(len(sts.LastDelays))
		varc := (smSq / float64(len(sts.LastDelays))) - (avgDly * avgDly)
		
		// jika delay sangat konsisten, mungkin uji coba / auto-pump
		if varc < 0.01 && avgDly < 2.0 {
			typ = DataTypeTest
			cfd = math.Max(cfd, 0.6)
		}
	}

	// 4. update statistik online (hanya jika data semi-valid)
	if typ == DataTypeValid || (typ == DataTypeTest && cfd < 0.8) {
		sts.WeightCount++
		// algoritma welford untuk rata-rata/varians online
		oldAvg := sts.AvgWeight
		sts.AvgWeight += (float64(wgt) - sts.AvgWeight) / float64(sts.WeightCount)
		sts.M2 += (float64(wgt) - oldAvg) * (float64(wgt) - sts.AvgWeight)
		
		// perlahan kurangi pengaruh data lama jika count terlalu tinggi
		if sts.WeightCount > sts.HistoryLength {
			sts.WeightCount = sts.HistoryLength
		}
	}

	return typ, cfd
}
