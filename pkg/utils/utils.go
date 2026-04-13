// Package utils provides generic helper functions shared across the kubeops-agent codebase.
package utils

func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// IFTernary returns trueVal when condition is true, falseVal otherwise.
// It is a generic inline ternary for cases where an if-expression improves readability.
func IFTernary[T any](condition bool, trueVal T, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
