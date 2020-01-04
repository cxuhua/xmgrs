package util

//移除重复的字符串
func RemoveRepeat(vs []string) []string {
	ms := map[string]bool{}
	for _, v := range vs {
		ms[v] = true
	}
	vs = []string{}
	for k, _ := range ms {
		vs = append(vs, k)
	}
	return vs
}
