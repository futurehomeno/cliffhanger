package utils

func Ptr[T any](val T) *T {
	return &val
}

func SliceContains[T comparable](value T, array []T) bool {
	for i := range array {
		if value == array[i] {
			return true
		}
	}

	return false
}
