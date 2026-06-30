package mdcw

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	DataTypeValid = "VALID"
	DataTypeIsen  = "ISEN"
	DataTypeSpam  = "SPAM"
	DataTypeTest  = "TEST"
)

type MachineStats struct {
	LastTs        time.Time
	WeightSum     float64
	WeightCount   int
	AvgWeight     float64
	M2            float64
	LastDelays    []float64
	HistoryLength int
}

var (
	machineStates   = make(map[string]*MachineStats)
	machineStatesMu sync.Mutex
)

func AnalyzeRecord(pfx string, wgt int) (string, float64) {
	machineStatesMu.Lock()
	defer machineStatesMu.Unlock()

	sts, ada := machineStates[pfx]
	wkt := time.Now()

	if !ada {
		machineStates[pfx] = &MachineStats{
			LastTs:        wkt,
			AvgWeight:     float64(wgt),
			WeightSum:     float64(wgt),
			WeightCount:   1,
			HistoryLength: 50,
		}
		return DataTypeValid, 0.7
	}

	dly := wkt.Sub(sts.LastTs).Seconds()
	sts.LastTs = wkt

	lwr := sts.AvgWeight * 0.4
	upr := sts.AvgWeight * 1.6

	typ := DataTypeValid
	cfd := 1.0

	if dly < 0.3 {
		typ = DataTypeSpam
		cfd = 0.9
	} else if wgt == 0 {
		typ = DataTypeIsen
		cfd = 1.0
	} else if float64(wgt) < lwr || float64(wgt) > upr {
		typ = DataTypeIsen
		cfd = 0.8

		if wgt > 0 && sts.AvgWeight > 0 {
			rt := float64(wgt) / sts.AvgWeight
			if math.Abs(rt-0.5) < 0.05 || math.Abs(rt-2.0) < 0.05 {
				typ = DataTypeTest
				cfd = 0.7
			}
		}
	}

	sts.LastDelays = append(sts.LastDelays, dly)
	if len(sts.LastDelays) > 5 {
		sts.LastDelays = sts.LastDelays[1:]

		var sm, smSq float64
		for _, d := range sts.LastDelays {
			sm += d
			smSq += d * d
		}
		avgDly := sm / float64(len(sts.LastDelays))
		varc := (smSq / float64(len(sts.LastDelays))) - (avgDly * avgDly)

		if varc < 0.01 && avgDly < 2.0 {
			typ = DataTypeTest
			cfd = math.Max(cfd, 0.6)
		}
	}

	if typ == DataTypeValid || (typ == DataTypeTest && cfd < 0.8) {
		sts.WeightCount++
		oldAvg := sts.AvgWeight
		sts.AvgWeight += (float64(wgt) - sts.AvgWeight) / float64(sts.WeightCount)
		sts.M2 += (float64(wgt) - oldAvg) * (float64(wgt) - sts.AvgWeight)

		if sts.WeightCount > sts.HistoryLength {
			sts.WeightCount = sts.HistoryLength
		}
	}

	return typ, cfd
}

func NormalizeRecord(prf string, reg5 int, reg114 int) (string, int, int) {
	if prf == "" {
		return prf, reg5, reg114
	}

	if reg114 < 0 {
		reg114 = -reg114
	}

	prfNorm := strings.ToUpper(strings.ReplaceAll(prf, " ", ""))

	if prfNorm == "MDCW" || strings.Contains(prfNorm, "TEST") || strings.Contains(prfNorm, "LINE2") || strings.Contains(prfNorm, "CEK") {
		return "IGNORE_RECORD", reg5, reg114
	}

	if strings.Contains(prfNorm, "MDCW1") || strings.Contains(prfNorm, "(UK)") {
		prf = "MDCW1 (UK)"
		if reg5 != 8201 {
			if reg114 < 8710 {
				reg5 = 25
			} else if reg114 > 9520 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW2") || strings.Contains(prfNorm, "SIOMAY") {
		prf = "MDCW2 (Siomay)"
		if reg5 != 8201 {
			if reg114 < 7040 {
				reg5 = 25
			} else if reg114 > 7540 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW3") || strings.Contains(prfNorm, "PENTOL") {
		prf = "MDCW3 (Pentol)"
		if reg5 != 8201 {
			if reg114 < 5840 {
				reg5 = 25
			} else if reg114 > 6340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW4") || strings.Contains(prfNorm, "AP") {
		prf = "MDCW4 (AP)"
		if reg5 != 8201 {
			if reg114 < 14940 {
				reg5 = 25
			} else if reg114 > 15660 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW5") || strings.Contains(prfNorm, "ACIN") {
		prf = "MDCW5 (ACIN)"
		if reg5 != 8201 {
			if reg114 < 10100 {
				reg5 = 25
			} else if reg114 > 10270 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW6") || strings.Contains(prfNorm, "LUMPIA") {
		prf = "MDCW6 (Lumpia)"
		if reg5 != 8201 {
			if reg114 < 3080 {
				reg5 = 25
			} else if reg114 > 3340 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW7") || strings.Contains(prfNorm, "KULIT") || strings.Contains(prfNorm, "KERUPUK") {
		prf = "MDCW7 (Kulit/Kerupuk)"
		if reg5 != 8201 {
			if reg114 < 9830 {
				reg5 = 25
			} else if reg114 > 10870 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW8") {
		prf = "MDCW8 (Mie)"
		if reg5 != 8201 {
			if reg114 < 9690 {
				reg5 = 25
			} else if reg114 > 10710 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW9") {
		prf = "MDCW9 (Mie)"
		if reg5 != 8201 {
			if reg114 < 9690 {
				reg5 = 25
			} else if reg114 > 10710 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MIE") {
		prf = "MDCW8 (Mie)"
		if reg5 != 8201 {
			if reg114 < 9690 {
				reg5 = 25
			} else if reg114 > 10710 {
				reg5 = 73
			} else {
				reg5 = 41
			}
		}
	} else if strings.Contains(prfNorm, "MDCW") {
		prf = prfNorm
		if reg5 != 8201 && (reg5 == 9 || reg5 == 90 || reg5 == 8 || reg5 == 0) && reg114 > 0 {
			reg5 = 41
		}
	} else {
		prf = strings.ToUpper(strings.TrimSpace(prf))
	}
	return prf, reg5, reg114
}

func MatchStatusFilter(fltSts string, reg5 int) bool {
	if fltSts == "" || fltSts == "all" {
		return true
	}
	switch fltSts {
	case "ok":
		return reg5 == 41 || reg5 == 521 || reg5 == 553
	case "idle":
		return reg5 == 9 || reg5 == 90
	case "metal":
		return reg5 == 8201
	case "under":
		return reg5 == 25
	case "over":
		return reg5 == 73
	case "mati":
		return reg5 == 8
	case "unknown":
		return reg5 != 8 && reg5 != 9 && reg5 != 90 && reg5 != 41 && reg5 != 521 && reg5 != 553 && reg5 != 8201 && reg5 != 25 && reg5 != 73
	default:
		return fmt.Sprintf("%d", reg5) == fltSts
	}
}
