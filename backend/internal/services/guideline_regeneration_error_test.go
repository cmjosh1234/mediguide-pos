package services

import (
	"strings"
	"testing"
)

func TestRegenerationTimeoutErrorIsActionableAndSafe(t *testing.T) {
	for _, raw := range []string{"timed out", "ReadTimeout", "Ollama embedding request timed out: http://private-host:11434"} {
		message := editorSafeRegenerationError(raw, "embeddings")
		if !strings.Contains(message, "Search indexing timed out") || !strings.Contains(message, "revision is unchanged") || !strings.Contains(message, "server capacity") {
			t.Fatalf("unexpected timeout guidance for %q: %q", raw, message)
		}
		if strings.Contains(message, "private-host") {
			t.Fatal("editor error exposed internal infrastructure")
		}
	}
	if strings.Contains(editorSafeRegenerationError("timed out", "downloading"), "Search indexing timed out") {
		t.Fatal("download timeout was misclassified as an embedding timeout")
	}
}
