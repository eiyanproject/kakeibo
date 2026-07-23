package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

type ParsedRow struct {
	Date        time.Time
	Merchant    string
	AmountMinor int64
}

type Mapping struct {
	DateCol    string
	DescCol    string
	AmountCol  string
	DateLayout string
	Invert     bool
	HasHeader  bool
}

// ReadCSV returns the header row and all rows (including the header row itself at index 0).
func ReadCSV(r io.Reader) (headers []string, rows [][]string, err error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	all, err := cr.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("empty csv")
	}
	return all[0], all, nil
}

func colIndex(headers []string, name string) (int, error) {
	for i, h := range headers {
		if strings.TrimSpace(h) == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("column %q not found", name)
}

// ParseAmount parses a money string like "1,650.00" or "¥1,650" into minor units (value * 100).
func ParseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "¥", "")
	s = strings.ReplaceAll(s, "円", "")
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f * 100)), nil
}

func ParseRows(headers []string, rows [][]string, m Mapping) ([]ParsedRow, error) {
	dateIdx, err := colIndex(headers, m.DateCol)
	if err != nil {
		return nil, err
	}
	descIdx, err := colIndex(headers, m.DescCol)
	if err != nil {
		return nil, err
	}
	amtIdx, err := colIndex(headers, m.AmountCol)
	if err != nil {
		return nil, err
	}

	start := 0
	if m.HasHeader {
		start = 1
	}
	layout := m.DateLayout
	if layout == "" {
		layout = "2006/01/02"
	}

	var out []ParsedRow
	for i := start; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= dateIdx || len(row) <= descIdx || len(row) <= amtIdx {
			continue
		}
		dateStr := strings.TrimSpace(row[dateIdx])
		if dateStr == "" {
			continue
		}
		d, err := time.Parse(layout, dateStr)
		if err != nil {
			return nil, fmt.Errorf("row %d: bad date %q: %w", i+1, dateStr, err)
		}
		amt, err := ParseAmount(row[amtIdx])
		if err != nil {
			return nil, fmt.Errorf("row %d: bad amount %q: %w", i+1, row[amtIdx], err)
		}
		if m.Invert {
			amt = -amt
		}
		out = append(out, ParsedRow{
			Date:        d,
			Merchant:    strings.TrimSpace(row[descIdx]),
			AmountMinor: amt,
		})
	}
	return out, nil
}

// DedupHash identifies a transaction independent of which import batch it came from,
// so re-uploading the same statement never creates duplicates.
func DedupHash(d time.Time, merchant string, amountMinor int64) string {
	return fmt.Sprintf("%s|%s|%d", d.Format("2006-01-02"), strings.ToLower(strings.TrimSpace(merchant)), amountMinor)
}
