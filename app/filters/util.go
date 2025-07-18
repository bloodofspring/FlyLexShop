package filters

import (
	"strings"
)

func ParseCallbackData(s string) map[string]string {
	res := make(map[string]string, 0)
	if len(strings.Split(s, "?")) != 2 {
		return res
	}
	params := strings.Trim(strings.Split(s, "?")[1], " ")

	for _, p := range strings.Split(params, "&") {
		if len(strings.Split(p, "=")) != 2 {
			continue
		}
		key := strings.Split(p, "=")[0]
		value := strings.Split(p, "=")[1]

		res[strings.Trim(key, " ")] = strings.Trim(value, " ")
	}

	return res
}
