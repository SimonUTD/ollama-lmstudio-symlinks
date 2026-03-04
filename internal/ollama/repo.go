package ollama

import "strings"

const defaultNamespace = "library"

func RepoForCLI(id ModelID) string {
	ns := strings.TrimSpace(id.Namespace)
	repo := strings.TrimSpace(id.Repository)
	if ns == "" || ns == defaultNamespace {
		return repo
	}
	if repo == "" {
		return ns
	}
	return ns + "/" + repo
}

