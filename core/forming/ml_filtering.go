package main

import (
	"math"
	"sync"
	"time"
)

// DataType constants
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

// AnalyzeRecord performs lightweight ML filtering on incoming production data
func AnalyzeRecord(prefix string, weight int) (string, float64) {
	machineStatesMu.Lock()
	defer machineStatesMu.Unlock()

	stats, exists := machineStates[prefix]
	now := time.Now()

	if !exists {
		// Initialize state for new machine
		machineStates[prefix] = &MachineStats{
			LastTs:        now,
			AvgWeight:     float64(weight),
			WeightSum:     float64(weight),
			WeightCount:   1,
			HistoryLength: 50, // Keep rolling stats for 50 records
		}
		return DataTypeValid, 0.7 // New machine, assume valid but medium confidence
	}

	// 1. Calculate Delay (Spam Check)
	delay := now.Sub(stats.LastTs).Seconds()
	stats.LastTs = now

	// 2. Weight Anomaly (Isen/Test Check)
	// We use the 60-80% tolerance (0.6 to 1.6 factor) mentioned by user
	lowerBound := stats.AvgWeight * 0.4 // 60% tolerance below
	upperBound := stats.AvgWeight * 1.6 // 60% tolerance above

	dataType := DataTypeValid
	confidence := 1.0

	// SPAM CHECK: Very high frequency
	if delay < 0.3 {
		dataType = DataTypeSpam
		confidence = 0.9
	} else if weight == 0 {
		dataType = DataTypeIsen
		confidence = 1.0
	} else if float64(weight) < lowerBound || float64(weight) > upperBound {
		// Out of tolerance
		dataType = DataTypeIsen
		confidence = 0.8
		
		// If it's a very specific outlier (e.g. exactly 1234 or half the avg), might be TEST
		if weight > 0 && stats.AvgWeight > 0 {
			ratio := float64(weight) / stats.AvgWeight
			if math.Abs(ratio-0.5) < 0.05 || math.Abs(ratio-2.0) < 0.05 {
				dataType = DataTypeTest
				confidence = 0.7
			}
		}
	}

	// 3. Test Pattern Detection
	// Check if this frequency is "too consistent" (human operator testing)
	stats.LastDelays = append(stats.LastDelays, delay)
	if len(stats.LastDelays) > 5 {
		stats.LastDelays = stats.LastDelays[1:]
		
		// Variance in delays
		var sum, sumSq float64
		for _, d := range stats.LastDelays {
			sum += d
			sumSq += d * d
		}
		avgDelay := sum / float64(len(stats.LastDelays))
		variance := (sumSq / float64(len(stats.LastDelays))) - (avgDelay * avgDelay)
		
		// If delay is very consistent, it might be a test/auto-pumping
		if variance < 0.01 && avgDelay < 2.0 {
			dataType = DataTypeTest
			confidence = math.Max(confidence, 0.6)
		}
	}

	// 4. Online Learning: Update Stats (only if semi-valid)
	if dataType == DataTypeValid || (dataType == DataTypeTest && confidence < 0.8) {
		stats.WeightCount++
		// Welford's algorithm for online mean/variance
		oldAvg := stats.AvgWeight
		stats.AvgWeight += (float64(weight) - stats.AvgWeight) / float64(stats.WeightCount)
		stats.M2 += (float64(weight) - oldAvg) * (float64(weight) - stats.AvgWeight)
		
		// Gradually decay influence of old data if count gets too high
		if stats.WeightCount > stats.HistoryLength {
			stats.WeightCount = stats.HistoryLength
		}
	}

	return dataType, confidence
}
