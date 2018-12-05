package sliceutil

// Go really should have generics..

func ContainsString(items []string, item string) bool {
	for _, candidate := range items {
		if candidate == item {
			return true
		}
	}

	return false
}

func ContainsInt(items []int, item int) bool {
	for _, candidate := range items {
		if candidate == item {
			return true
		}
	}

	return false
}

func FilterInt(items []int, fn func(item int) bool) []int {
	altered := []int{}

	for _, item := range items {
		if fn(item) {
			altered = append(altered, item)
		}
	}

	return altered
}

func ReverseStringSlice(input []string) []string {
	maxIdx := len(input) - 1
	ret := make([]string, maxIdx+1)

	for i := maxIdx; i >= 0; i-- {
		ret[maxIdx-i] = input[i]
	}

	return ret
}
