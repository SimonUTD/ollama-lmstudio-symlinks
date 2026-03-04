package ollama

import (
	"fmt"
	"strings"
)

const maxModelNamePartLen = 80

func ValidateModelName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("Ollama 名称为空")
	}
	if strings.ContainsAny(name, " \t\r\n") {
		return fmt.Errorf("Ollama 名称包含空白字符")
	}
	if strings.Count(name, ":") > 1 {
		return fmt.Errorf("Ollama 名称最多只能包含一个 ':'（用于分隔 tag）")
	}

	repo, tag := splitRepoAndTag(name)
	if err := validateRepo(repo); err != nil {
		return err
	}
	if strings.Contains(name, ":") && tag == "" {
		return fmt.Errorf("Ollama tag 不能为空")
	}
	if tag == "" {
		return nil
	}
	return validatePart(tag, "tag")
}

func splitRepoAndTag(name string) (string, string) {
	last := strings.LastIndex(name, ":")
	if last == -1 {
		return name, ""
	}
	return name[:last], name[last+1:]
}

func validateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("Ollama 仓库名为空")
	}
	if strings.HasPrefix(repo, "/") || strings.HasSuffix(repo, "/") {
		return fmt.Errorf("Ollama 仓库名不能以 '/' 开头或结尾")
	}

	parts := strings.Split(repo, "/")
	for _, p := range parts {
		if err := validatePart(p, "repo"); err != nil {
			return err
		}
	}
	return nil
}

func validatePart(part string, kind string) error {
	if part == "" {
		return fmt.Errorf("Ollama %s 段不能为空", kind)
	}
	if len(part) > maxModelNamePartLen {
		return fmt.Errorf("Ollama %s 段过长（>%d）：%d", kind, maxModelNamePartLen, len(part))
	}

	switch part[0] {
	case '.', '-':
		return fmt.Errorf("Ollama %s 段不能以 '.' 或 '-' 开头：%q", kind, part)
	}

	for _, r := range part {
		if !isAllowedNameRune(r) {
			return fmt.Errorf("Ollama %s 段包含非法字符 %q：%q", kind, r, part)
		}
	}
	return nil
}

func isAllowedNameRune(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case '-', '_', '.':
		return true
	default:
		return false
	}
}
