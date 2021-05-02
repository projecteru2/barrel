package utils

func FilterStringArray(array []string, filter func(string) bool) (result []string) {
	for _, value := range array {
		if filter(value) {
			result = append(result, value)
		}
	}
	return
}
