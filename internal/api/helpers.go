package api

func maskPassword(p string) string {
	if p == "" {
		return ""
	}
	return "********"
}
