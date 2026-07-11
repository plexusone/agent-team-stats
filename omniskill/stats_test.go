package omniskill

import (
	"testing"

	"github.com/plexusone/agent-team-stats/pkg/config"
)

func TestStatsSkill_Name(t *testing.T) {
	skill := New(Config{
		AppConfig: &config.Config{},
	})

	if skill.Name() != "statistics" {
		t.Errorf("expected name 'statistics', got %q", skill.Name())
	}
}

func TestStatsSkill_Description(t *testing.T) {
	skill := New(Config{
		AppConfig: &config.Config{},
	})

	desc := skill.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestStatsSkill_Tools(t *testing.T) {
	skill := New(Config{
		AppConfig: &config.Config{},
	})

	tools := skill.Tools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	expectedTools := []string{"research_statistics", "get_status"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestStatsSkill_Tools_Parameters(t *testing.T) {
	skill := New(Config{
		AppConfig: &config.Config{},
	})

	tools := skill.Tools()

	// Find research_statistics tool and check its parameters
	found := false
	for _, tool := range tools {
		if tool.Name() == "research_statistics" {
			found = true
			params := tool.Parameters()
			// Check required parameters exist
			if _, ok := params["topic"]; !ok {
				t.Error("research_statistics missing 'topic' parameter")
			}
			if _, ok := params["min_verified"]; !ok {
				t.Error("research_statistics missing 'min_verified' parameter")
			}
			if _, ok := params["max_candidates"]; !ok {
				t.Error("research_statistics missing 'max_candidates' parameter")
			}
			if _, ok := params["reputable_only"]; !ok {
				t.Error("research_statistics missing 'reputable_only' parameter")
			}
			break
		}
	}

	if !found {
		t.Error("research_statistics tool not found")
	}
}

func TestStatsSkill_Close(t *testing.T) {
	skill := New(Config{
		AppConfig: &config.Config{},
	})

	// Close should not error even without Init
	if err := skill.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
