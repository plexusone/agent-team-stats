package synthesis

import (
	"testing"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/models"
)

func TestWorker_Properties(t *testing.T) {
	w := New(Config{})

	if w.ID() != "synthesis" {
		t.Errorf("expected ID 'synthesis', got %q", w.ID())
	}

	if w.Type() != "synthesis" {
		t.Errorf("expected Type 'synthesis', got %q", w.Type())
	}

	if w.Version() != "2.0.0" {
		t.Errorf("expected Version '2.0.0', got %q", w.Version())
	}
}

func TestWorker_CustomConfig(t *testing.T) {
	w := New(Config{
		WorkerConfig: worker.WorkerConfig{
			ID:      "custom-synthesis",
			Type:    "custom-type",
			Version: "3.0.0",
		},
	})

	if w.ID() != "custom-synthesis" {
		t.Errorf("expected ID 'custom-synthesis', got %q", w.ID())
	}
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON array",
			input:    `[{"name": "test", "value": 1}]`,
			expected: `[{"name": "test", "value": 1}]`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n[{\"name\": \"test\", \"value\": 1}]\n```",
			expected: `[{"name": "test", "value": 1}]`,
		},
		{
			name:     "JSON with leading text",
			input:    "Here are the statistics:\n[{\"name\": \"test\"}]",
			expected: `[{"name": "test"}]`,
		},
		{
			name:     "JSON with trailing text",
			input:    "[{\"name\": \"test\"}]\nThat's all the statistics.",
			expected: `[{"name": "test"}]`,
		},
		{
			name:     "empty array",
			input:    "[]",
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSONFromMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertSearchResults(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantLen int
		wantErr bool
	}{
		{
			name: "slice of maps",
			input: []any{
				map[string]any{
					"url":     "https://example.com",
					"title":   "Test Title",
					"snippet": "Test snippet",
					"domain":  "example.com",
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "direct slice",
			input: []models.SearchResult{
				{URL: "https://example.com", Title: "Test", Snippet: "Snip", Domain: "example.com"},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "not a slice",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertSearchResults(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertSearchResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("convertSearchResults() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"name":   "test",
		"number": 42,
		"nil":    nil,
	}

	if got := getString(m, "name"); got != "test" {
		t.Errorf("getString(name) = %q, want 'test'", got)
	}

	if got := getString(m, "number"); got != "" {
		t.Errorf("getString(number) = %q, want ''", got)
	}

	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString(missing) = %q, want ''", got)
	}
}
