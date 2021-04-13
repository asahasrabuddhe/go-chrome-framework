package chrome

// String returns a pointer to the given string
func String(str string) *string {
	return &str
}

// StringValue returns the value of the string pointer passed in or "" if the pointer is nil
func StringValue(str *string) string {
	if str != nil {
		return *str
	}
	return ""
}

// StringSlice returns slice of string pointers for given slice of string
func StringSlice(slice []string) []*string {
	value := make([]*string, len(slice))
	for i := 0; i < len(slice); i++ {
		value[i] = String(slice[i])
	}
	return value
}

// StringValueSlice returns slice of string values for given slice of string pointer. "" is returned if
// any pointer in the slice is nil
func StringValueSlice(slice []*string) []string {
	value := make([]string, len(slice))
	for i := 0; i < len(slice); i++ {
		if slice[i] != nil {
			value[i] = StringValue(slice[i])
		}
	}
	return value
}

// Int returns a pointer to the given integer
func Int(number int) *int {
	return &number
}

// IntValue returns the value of the integer pointer or 0 if the pointer is nil
func IntValue(number *int) int {
	if number != nil {
		return *number
	}
	return 0
}
