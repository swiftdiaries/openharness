package cost

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Record struct {
	Timestamp    time.Time
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	SessionID    string
}

type Tracker struct {
	filePath     string
	dailyLimit   float64
	monthlyLimit float64
	mu           sync.Mutex
}

func NewTracker(dataDir string, dailyLimit, monthlyLimit float64) (*Tracker, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &Tracker{
		filePath:     filepath.Join(dataDir, "cost_log.csv"),
		dailyLimit:   dailyLimit,
		monthlyLimit: monthlyLimit,
	}, nil
}

func (t *Tracker) Record(r Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	f, err := os.OpenFile(t.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open cost log: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	return w.Write([]string{
		r.Timestamp.Format(time.RFC3339),
		r.Provider,
		r.Model,
		strconv.Itoa(r.InputTokens),
		strconv.Itoa(r.OutputTokens),
		fmt.Sprintf("%.6f", r.CostUSD),
		r.SessionID,
	})
}

func (t *Tracker) CheckBudget() (bool, string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	records, err := t.readRecords()
	if err != nil {
		// If we can't read, allow but warn
		return true, ""
	}

	now := time.Now()
	var dailyTotal, monthlyTotal float64

	for _, r := range records {
		if r.Timestamp.Year() == now.Year() && r.Timestamp.Month() == now.Month() {
			monthlyTotal += r.CostUSD
			if r.Timestamp.Day() == now.Day() {
				dailyTotal += r.CostUSD
			}
		}
	}

	if t.dailyLimit > 0 && dailyTotal >= t.dailyLimit {
		return false, fmt.Sprintf("daily budget exceeded: $%.2f / $%.2f", dailyTotal, t.dailyLimit)
	}
	if t.monthlyLimit > 0 && monthlyTotal >= t.monthlyLimit {
		return false, fmt.Sprintf("monthly budget exceeded: $%.2f / $%.2f", monthlyTotal, t.monthlyLimit)
	}
	return true, ""
}

func (t *Tracker) readRecords() ([]Record, error) {
	f, err := os.Open(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var records []Record
	for _, row := range rows {
		if len(row) < 7 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, row[0])
		if err != nil {
			continue
		}
		input, _ := strconv.Atoi(row[3])
		output, _ := strconv.Atoi(row[4])
		cost, _ := strconv.ParseFloat(row[5], 64)
		records = append(records, Record{
			Timestamp:    ts,
			Provider:     row[1],
			Model:        row[2],
			InputTokens:  input,
			OutputTokens: output,
			CostUSD:      cost,
			SessionID:    row[6],
		})
	}
	return records, nil
}
