package ollama

import "testing"

func TestRepoForCLI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		id   ModelID
		want string
	}{
		{name: "library simple", id: ModelID{Namespace: "library", Repository: "llama3"}, want: "llama3"},
		{name: "library nested", id: ModelID{Namespace: "library", Repository: "myuser/my-model"}, want: "myuser/my-model"},
		{name: "named namespace", id: ModelID{Namespace: "unsloth", Repository: "qwen3"}, want: "unsloth/qwen3"},
		{name: "named namespace nested", id: ModelID{Namespace: "teichai", Repository: "glm-4.7-flash"}, want: "teichai/glm-4.7-flash"},
	}

	for _, tc := range cases {
		if got := RepoForCLI(tc.id); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

