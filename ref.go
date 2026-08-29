package formula

import "fmt"

func splitRef(ref string) (letters, digits string, err error) {
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(ref) {
		return "", "", fmt.Errorf("formula: %q is not a cell reference", ref)
	}
	for j := i; j < len(ref); j++ {
		if ref[j] < '0' || ref[j] > '9' {
			return "", "", fmt.Errorf("formula: %q is not a cell reference", ref)
		}
	}
	return ref[:i], ref[i:], nil
}

func looksLikeRef(s string) bool {
	_, _, err := splitRef(s)
	return err == nil
}
