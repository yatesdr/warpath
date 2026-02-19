package plc

import (
	"bufio"
	"io"
	"strings"
)

// SSERawEvent represents a single parsed SSE event from the wire.
type SSERawEvent struct {
	Event string
	Data  string
	ID    string
}

// SSEReader reads SSE events from an io.Reader using a bufio.Scanner.
type SSEReader struct {
	scanner *bufio.Scanner
}

// NewSSEReader creates a new SSE stream reader.
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{scanner: bufio.NewScanner(r)}
}

// Next returns the next complete SSE event, or an error (io.EOF at end of stream).
// It blocks until a full event is available.
func (s *SSEReader) Next() (SSERawEvent, error) {
	var ev SSERawEvent
	var dataParts []string
	hasFields := false

	for s.scanner.Scan() {
		line := s.scanner.Text()

		// Blank line dispatches the event
		if line == "" {
			if hasFields {
				ev.Data = strings.Join(dataParts, "\n")
				return ev, nil
			}
			continue
		}

		// Comment lines (starting with ':') are ignored
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Split on first ':'
		field := line
		value := ""
		if idx := strings.Index(line, ":"); idx >= 0 {
			field = line[:idx]
			value = line[idx+1:]
			// Strip single leading space from value per SSE spec
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
		}

		switch field {
		case "event":
			ev.Event = value
			hasFields = true
		case "data":
			dataParts = append(dataParts, value)
			hasFields = true
		case "id":
			ev.ID = value
			hasFields = true
		}
	}

	if err := s.scanner.Err(); err != nil {
		return SSERawEvent{}, err
	}

	// EOF with accumulated fields: dispatch final event
	if hasFields {
		ev.Data = strings.Join(dataParts, "\n")
		return ev, nil
	}

	return SSERawEvent{}, io.EOF
}
