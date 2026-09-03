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

// colIndex converts a column letter sequence (A, B, ..., Z, AA, AB, ...) to a
// 1-based number, matching spreadsheet column ordering.
func colIndex(letters string) int {
	n := 0
	for i := 0; i < len(letters); i++ {
		n = n*26 + int(letters[i]-'A'+1)
	}
	return n
}

// colName is the inverse of colIndex.
func colName(n int) string {
	var buf []byte
	for n > 0 {
		n--
		buf = append([]byte{byte('A' + n%26)}, buf...)
		n /= 26
	}
	return string(buf)
}
