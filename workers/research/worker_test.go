package research

import (
	"testing"

	worker "github.com/plexusone/omniagent-worker"
)

func TestWorker_Properties(t *testing.T) {
	w := New(Config{})

	if w.ID() != "research" {
		t.Errorf("expected ID 'research', got %q", w.ID())
	}

	if w.Type() != "research" {
		t.Errorf("expected Type 'research', got %q", w.Type())
	}

	if w.Version() != "2.0.0" {
		t.Errorf("expected Version '2.0.0', got %q", w.Version())
	}
}

func TestWorker_CustomConfig(t *testing.T) {
	w := New(Config{
		WorkerConfig: worker.WorkerConfig{
			ID:      "custom-research",
			Type:    "custom-type",
			Version: "3.0.0",
		},
	})

	if w.ID() != "custom-research" {
		t.Errorf("expected ID 'custom-research', got %q", w.ID())
	}

	if w.Type() != "custom-type" {
		t.Errorf("expected Type 'custom-type', got %q", w.Type())
	}

	if w.Version() != "3.0.0" {
		t.Errorf("expected Version '3.0.0', got %q", w.Version())
	}
}

func TestIsReputableSource(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"www.cdc.gov", true},
		{"climate.nasa.gov", true},
		{"www.who.int", true},
		{"www.worldbank.org", true},
		{"www.pewresearch.org", true},
		{"www.nature.com", true},
		{"www.science.org", true},
		{"harvard.edu", true},
		{"www.stanford.edu", true},
		{"random-blog.com", false},
		{"news-site.net", false},
		{"example.org", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isReputableSource(tt.domain)
			if result != tt.expected {
				t.Errorf("isReputableSource(%q) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}
