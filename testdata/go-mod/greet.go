package gomod

// Greet returns a greeting for name.
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return "hello, " + name
}
