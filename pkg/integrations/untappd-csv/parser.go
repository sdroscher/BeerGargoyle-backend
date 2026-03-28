// Package untappdcsv parses and processes Untappd CSV checkin exports.
package untappdcsv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ErrMissingColumn is returned when a required CSV column is absent from the header.
var ErrMissingColumn = errors.New("missing required column")

const csvDateFormat = "2006-01-02 15:04:05"

// Row represents a single parsed checkin row from an Untappd CSV export.
type Row struct {
	BeerName    string
	BreweryName string
	RatingScore float64
	CreatedAt   time.Time
	Comment     string
	ServingType string
	CheckinID   uint64
	BeerID      uint64 // Untappd beer ID ("bid" column)
}

// Parse reads an Untappd CSV export (possibly with a UTF-8 BOM) and returns all parseable rows.
// Rows with malformed dates or missing checkin_id are silently skipped.
func Parse(r io.Reader) ([]Row, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	// Strip UTF-8 BOM from first field name if present.
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	colIdx := make(map[string]int, len(header))
	for i, name := range header {
		colIdx[name] = i
	}

	required := []string{"beer_name", "brewery_name", "rating_score", "created_at", "comment", "serving_type", "checkin_id", "bid"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingColumn, col)
		}
	}

	var rows []Row

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("reading CSV row: %w", err)
		}

		row, ok := parseRow(record, colIdx)
		if !ok {
			continue
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func parseRow(record []string, colIdx map[string]int) (Row, bool) {
	get := func(col string) string {
		idx, ok := colIdx[col]
		if !ok || idx >= len(record) {
			return ""
		}

		return strings.TrimSpace(record[idx])
	}

	createdAtStr := get("created_at")

	createdAt, err := time.ParseInLocation(csvDateFormat, createdAtStr, time.UTC)
	if err != nil {
		return Row{}, false
	}

	checkinIDStr := get("checkin_id")

	checkinID, err := strconv.ParseUint(checkinIDStr, 10, 64)
	if err != nil {
		return Row{}, false
	}

	beerID, _ := strconv.ParseUint(get("bid"), 10, 64)

	var rating float64
	if s := get("rating_score"); s != "" {
		rating, _ = strconv.ParseFloat(s, 64)
	}

	return Row{
		BeerName:    get("beer_name"),
		BreweryName: get("brewery_name"),
		RatingScore: rating,
		CreatedAt:   createdAt,
		Comment:     get("comment"),
		ServingType: get("serving_type"),
		CheckinID:   checkinID,
		BeerID:      beerID,
	}, true
}

// NameKey returns the normalised lookup key used in the beerIDByName map.
// Both the map builder and the row matcher must use this function.
func NameKey(beerName, breweryName string) string {
	return strings.ToLower(beerName) + "|" + strings.ToLower(breweryName)
}
