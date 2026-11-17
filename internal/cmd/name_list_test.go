package cmd_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.invalid/mcp-template-module-placeholder/internal/cmd"
)

func TestToolsListNameListBoundary(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("unrelated: super-secret-value\nenabled_tools:\n  - echo\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Run("empty env preserves yaml", func(t *testing.T) {
		t.Setenv("MCP_ENABLED_TOOLS", "")
		names, err := listToolNames(t, "--config", configPath)
		if err != nil {
			t.Fatalf("tools list: %v", err)
		}
		if strings.Join(names, ",") != "echo" {
			t.Fatalf("tools = %v, want echo", names)
		}
	})

	t.Run("blank env fails without leaking config", func(t *testing.T) {
		t.Setenv("MCP_ENABLED_TOOLS", " , ")
		_, err := listToolNames(t, "--config", configPath)
		if err == nil || !strings.Contains(err.Error(), "enabled_tools contains no names") {
			t.Fatalf("tools list error = %v", err)
		}
		if strings.Contains(err.Error(), "super-secret-value") {
			t.Fatalf("tools list error leaked config: %v", err)
		}
	})

	t.Run("changed flag lists trimmed names", func(t *testing.T) {
		names, err := listToolNames(t, "--enabled-tools", " echo , , ping ")
		if err != nil {
			t.Fatalf("tools list: %v", err)
		}
		if strings.Join(names, ",") != "echo,ping" {
			t.Fatalf("tools = %v, want echo,ping", names)
		}
	})

	t.Run("changed blank flag fails closed", func(t *testing.T) {
		_, err := listToolNames(t, "--enabled-tools", " , ")
		if err == nil || !strings.Contains(err.Error(), "enabled_tools contains no names") {
			t.Fatalf("tools list error = %v", err)
		}
	})

	t.Run("unknown name does not fail the list", func(t *testing.T) {
		names, err := listToolNames(t, "--enabled-tools", "missing")
		if err != nil {
			t.Fatalf("tools list: %v", err)
		}
		if len(names) != 0 {
			t.Fatalf("tools = %v, want none", names)
		}
	})
}

func listToolNames(t *testing.T, args ...string) ([]string, error) {
	t.Helper()
	var out bytes.Buffer
	root := cmd.NewRootCommand(cmd.IOStreams{Out: &out, ErrOut: &out})
	root.SetArgs(append([]string{"tools", "list", "--json"}, args...))
	if err := root.Execute(); err != nil {
		return nil, err
	}
	var listed []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal tools: %v\n%s", err, out.String())
	}
	names := make([]string, 0, len(listed))
	for _, tool := range listed {
		names = append(names, tool.Name)
	}
	return names, nil
}
