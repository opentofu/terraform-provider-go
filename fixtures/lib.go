package lib

func Hello(name *string) map[string]*string {
	return map[string]*string{
		"hello": name,
		"empty": nil,
	}
}
