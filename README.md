# cell-formula-engine

A small Go library for parsing and evaluating spreadsheet-style formulas —
the kind you'd type into a cell in Excel or Google Sheets — without
dragging in a full spreadsheet engine.

I wanted this for a reporting tool where users define derived columns like
`=(A1-B1)/B1` or `=SUM(A1,A2,A3)` and the values get recomputed whenever
the underlying data changes. Every general-purpose "formula parser" I found
either came bundled with a whole spreadsheet UI or pulled in a dozen
transitive dependencies for features I didn't need. This does one thing:
turn a formula string plus a map of cell values into a number.

## What it handles right now

- Arithmetic: `+ - * /`, parentheses, unary minus
- Cell references: `A1`, `B12`, `AA7`
- Function calls over a plain argument list: `SUM(...)`, `AVERAGE(...)`
- A leading `=` is optional, so you can feed it raw user input either way

Cell ranges (`A1:B3`) aren't supported yet — see Roadmap below.

## Usage

```go
package main

import (
	"fmt"

	formula "github.com/danielw382/cell-formula-engine"
)

func main() {
	sheet := formula.Sheet{
		"A1": 120,
		"A2": 80,
		"A3": 45,
	}

	total, err := formula.Evaluate("=SUM(A1,A2,A3)", sheet)
	if err != nil {
		panic(err)
	}
	fmt.Println(total) // 245

	avg, err := formula.Evaluate("AVERAGE(A1,A2,A3)", sheet)
	if err != nil {
		panic(err)
	}
	fmt.Println(avg) // 81.666...

	pct, err := formula.Evaluate("=(A1-A2)/A2*100", sheet)
	if err != nil {
		panic(err)
	}
	fmt.Println(pct) // 50
}
```

`formula.Sheet` is just `map[string]float64`. Referencing a cell that isn't
in the map is an evaluation error, not a zero — that's deliberate, since a
silent zero for a typo'd reference is exactly the kind of bug this library
is meant to catch before it reaches a report.

## Design

The package is a straightforward hand-written lexer, a recursive-descent
(precedence-climbing) parser, and a tree-walking evaluator. No reflection,
no code generation, no external grammar. `Evaluate` re-parses the formula
on every call; if you're evaluating the same formula against many sheets,
parse once yourself and cache the tree — that split just hasn't been
needed yet, so the API doesn't expose it.

## Roadmap

- Range support (`SUM(A1:B3)`)
- `MIN`, `MAX`, `COUNT`
- String and boolean values, comparison operators
- A parsed-formula type so repeated evaluation skips re-parsing
