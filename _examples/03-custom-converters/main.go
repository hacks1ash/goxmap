// Example 03: Custom converter functions
//
// This example demonstrates using `mapper:"func:FnName"` tags to delegate
// field conversion to custom functions. This is useful for:
//   - Converting time.Time to a formatted string
//   - Computing derived fields (e.g., full name from first + last)
//   - Applying domain-specific transformations
//
// To regenerate:
//
//	go generate ./...
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

//go:generate go run github.com/hacks1ash/goxmap -src Event -dst EventDTO

// Event is the domain model with rich Go types.
type Event struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Tags        string    `json:"tags"`
}

// EventDTO is the API response with string representations.
type EventDTO struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time" mapper:"func:FormatTime"`
	EndTime     string `json:"end_time" mapper:"func:FormatTime"`
	Tags        string `json:"tags" mapper:"func:NormalizeTags"`
}

// FormatTime converts a time.Time to an RFC3339 string.
// Referenced by the mapper:"func:FormatTime" tag on EventDTO fields.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// NormalizeTags lowercases and trims a comma-separated tag string.
// Referenced by the mapper:"func:NormalizeTags" tag on EventDTO.Tags.
func NormalizeTags(tags string) string {
	parts := strings.Split(tags, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ToLower(p))
	}
	return strings.Join(parts, ", ")
}

func main() {
	event := &Event{
		ID:          1,
		Title:       "Go Meetup",
		Description: "Monthly Go developer meetup",
		StartTime:   time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 6, 15, 20, 0, 0, 0, time.UTC),
		Tags:        "Go, Meetup, PROGRAMMING, Community",
	}

	dto := MapEventToEventDTO(event)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(dto); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
