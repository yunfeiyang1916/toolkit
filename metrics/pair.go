package metrics

import "strings"

type kvPairs [][2]string

func (kv kvPairs) Len() int {
	return len(kv)
}
func (kv kvPairs) Swap(i, j int) {
	kv[i], kv[j] = kv[j], kv[i]
}

func (kv kvPairs) Less(i, j int) bool {
	return strings.Compare(kv[i][0], kv[j][0]) < 0
}
