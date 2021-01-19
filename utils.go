package main

import (
	"path/filepath"
	"strings"
)

// GetNormalizedExtension ...
func GetNormalizedExtension(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

// GetFileType ...
func GetFileType(path string) string {
	return fileExtToType[GetNormalizedExtension(path)]
}

// IsHidden ...
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// Max for ints
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func bestUnit(size int64) (float32, string) {
	switch {
	case size > 100*1000*1000*1000:
		return 1000 * 1000 * 1000, "GB"
	case size > 100*1000*1000:
		return 1000 * 1000, "MB"
	case size > 100*1000:
		return 1000, "KB"
	default:
		return 1, "bytes"
	}
}
