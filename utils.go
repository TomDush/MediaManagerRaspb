package main

// strings.Join, but without empty values
func joinNotEmpty(values []string, separator string) string {
	var res string
	for _, val := range values {
		if val != "" {
			if res != "" {
				res += "/"
			}

			res += val
		}
	}

	return res
}