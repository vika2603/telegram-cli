package output

import "strconv"

// itoaOrBlank converts n to its decimal string representation.
// It returns "" when n is zero so table cells remain blank for missing values.
func itoaOrBlank(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// i64toaOrBlank converts n to its decimal string representation.
// It returns "" when n is zero so table cells remain blank for missing values.
func i64toaOrBlank(n int64) string {
	if n == 0 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}
