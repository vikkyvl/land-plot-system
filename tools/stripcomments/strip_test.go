package main

import "testing"

const source = `// Package sample is documented.
package sample

import _ "embed"

//go:embed schema.sql
var schema string

// Point is documented.
type Point struct {
	// X is the horizontal position.
	X int
	Y int // vertical position
}

// Sum is documented.
func Sum(values []int) int {
	// accumulate the total
	total := 0
	for _, value := range values {
		total += value // add
	}
	//nolint:gosec // directive is kept
	apply := func() {
		// inside a function literal
	}
	apply()
	return total
}
`

const want = `package sample

import _ "embed"

//go:embed schema.sql
var schema string

type Point struct {
	X int
	Y int
}

func Sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	//nolint:gosec // directive is kept
	apply := func() {
	}
	apply()
	return total
}
`

func TestStripRemovesEveryCommentButDirectives(t *testing.T) {
	got, count, err := stripComments("sample.go", []byte(source))
	if err != nil {
		t.Fatalf("strip() error = %v", err)
	}

	if count != 8 {
		t.Fatalf("removed %d comments, want 8", count)
	}

	if string(got) != want {
		t.Fatalf("strip() =\n%s\nwant\n%s", got, want)
	}
}

func TestStripLeavesCleanFileUnchanged(t *testing.T) {
	_, count, err := stripComments("sample.go", []byte(want))
	if err != nil || count != 0 {
		t.Fatalf("strip() count = %d, error = %v", count, err)
	}
}
