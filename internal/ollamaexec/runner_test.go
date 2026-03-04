package ollamaexec

import (
	"context"
	"strings"
	"testing"
)

func TestExecRunnerCreateFromGGUF_RejectsNewlinesInPath(t *testing.T) {
	t.Parallel()

	runner := ExecRunner{BinPath: "/does-not-exist", Host: "127.0.0.1:11434"}
	err := runner.CreateFromGGUF(context.Background(), "my-model", "bad\npath.gguf")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "换行") {
		t.Fatalf("expected error to mention newline, got: %v", err)
	}
}
