package util

import "strings"

import "strconv"

// IndentLines indents all the lines with spaces
func IndentLines(lines []string, char string, count int) []string {
	for i := range lines {
		lines[i] = strings.Repeat(char, count) + lines[i]
	}
	return lines
}

// MustParseInt uses Atoi to create int from
// the given argument, and panics if there is an error.
func MustParseInt(number string) int {
	i, err := strconv.Atoi(number)
	if err != nil {
		panic(err)
	}
	return i
}
