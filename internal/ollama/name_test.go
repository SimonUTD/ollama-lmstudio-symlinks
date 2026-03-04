package ollama

import (
	"strings"
	"testing"
)

func TestValidateModelName_AllowsValidNames(t *testing.T) {
	t.Parallel()

	cases := []string{
		"llama3",
		"AbC",
		"my_model",
		"my-model",
		"my.model",
		"user/model",
		"registry.ollama.ai/library/llama3:latest",
		"abc:_tag",
		"1abc:2tag",
	}
	for _, c := range cases {
		if err := ValidateModelName(c); err != nil {
			t.Fatalf("%q: expected ok, got %v", c, err)
		}
	}
}

func TestValidateModelName_RejectsInvalidNames(t *testing.T) {
	t.Parallel()

	repo81 := strings.Repeat("a", 81)
	tag81 := "abc:" + strings.Repeat("a", 81)

	cases := []string{
		"",
		"   ",
		"a b",
		".abc",
		"-abc",
		"a//b",
		"a/.b",
		"a/-b",
		"abc:.tag",
		"abc:-tag",
		"a+b",
		"a@b",
		"a:b:c",
		repo81,
		tag81,
		"abc:",
		":tag",
		"/abc",
		"abc/",
	}

	for _, c := range cases {
		if err := ValidateModelName(c); err == nil {
			t.Fatalf("%q: expected error, got nil", c)
		}
	}
}

