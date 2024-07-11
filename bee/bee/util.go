package bee

import "strings"

// 匹配通配符的辅助函数
func matchWildcard(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(parts) != len(pathParts) {
		return false
	}

	for i, part := range parts {
		if part == "*" {
			continue
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}
