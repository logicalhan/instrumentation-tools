/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package term_test

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"github.com/gdamore/tcell"

	"sigs.k8s.io/instrumentation-tools/promq/term"
)

type cellsMatcher struct {
	expected tcell.SimulationScreen
	contentsOnly bool
}
func (m *cellsMatcher) onScreenAsCells(contents term.Flushable) []tcell.SimCell {
	screen := m.onScreen(contents)
	cells, _, _ := screen.GetContents()
	return cells
}
func (m *cellsMatcher) onScreen(contents term.Flushable) tcell.SimulationScreen {
	screen := tcell.NewSimulationScreen("")
	screen.Init()
	screen.SetSize(m.expected.Size())
	contents.FlushTo(screen)
	screen.Show()

	return screen
}
func (m *cellsMatcher) Match(actual interface{}) (bool, error) {
	if m.expected == nil && actual == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <nnil>")
	}
	
	expectedCells, _, _ := m.expected.GetContents()

	var (
		actualCells []tcell.SimCell
	)
	switch actual := actual.(type) {
	case term.Flushable:
		actualCells = m.onScreenAsCells(actual)
	case tcell.SimulationScreen:
		actualCells, _, _ = actual.GetContents()
	default:
		return reflect.DeepEqual(expectedCells, actual), nil
	}
	if !m.contentsOnly {
		return reflect.DeepEqual(expectedCells, actualCells), nil
	}

	expectedRunes := make([]rune, 0, len(expectedCells))
	for _, cell := range expectedCells {
		expectedRunes = append(expectedRunes, cell.Runes...)
	}
	actualRunes := make([]rune, 0, len(actualCells))
	for _, cell := range actualCells {
		actualRunes = append(actualRunes, cell.Runes...)
	}

	return reflect.DeepEqual(expectedRunes, actualRunes), nil
}

func (m *cellsMatcher) FailureMessage(actual interface{}) string {
	var actualScreen tcell.SimulationScreen
	switch actual := actual.(type) {
	case term.Flushable:
		actualScreen = m.onScreen(actual)
	case tcell.SimulationScreen:
		actualScreen = actual
	default:
		return format.Message(actual, "to equal", displayCells(m.expected))
	}

	if m.contentsOnly {
		return format.Message("\n"+displayCells(actualScreen), "to equal (ignoring style)", "\n"+displayCells(m.expected))
	} else {
		return format.Message("\n"+displayCells(actualScreen), "to equal (including style, not shown)", "\n"+displayCells(m.expected))
	}
}

func (m *cellsMatcher) NegatedFailureMessage(actual interface{}) string {
	var actualScreen tcell.SimulationScreen
	switch actual := actual.(type) {
	case term.Flushable:
		actualScreen = m.onScreen(actual)
	case tcell.SimulationScreen:
		actualScreen = actual
	default:
		return format.Message(actual, "not to equal", displayCells(m.expected))
	}
	
	if m.contentsOnly {
		return format.Message("\n"+displayCells(actualScreen), "not to equal (ignoring style)", "\n"+displayCells(m.expected))
	} else {
		return format.Message("\n"+displayCells(actualScreen), "not to equal (including style, not shown)", "\n"+displayCells(m.expected))
	}
}

// displayCells displays the given cells w/o formatting as they'd be displayed
// on the screen (e.g. wrapped to width, etc).  Does not currently take into
// account character width (largely relevant for full-width vs half-width CJK)
func displayCells(screen tcell.SimulationScreen) string {
	cells, _, _ := screen.GetContents()
	screenCols, _ := screen.Size()

	var res []rune
	for i, cell := range cells {
		if i % screenCols == 0 && i != 0 {
			res = append(res, '\n')
		}
		if len(cell.Runes) != 0 {
			res = append(res, cell.Runes[0])
		}
	}

	return string(res)
}

// DisplayLike matches the given string to the contents to the actual screen,
// ignoring styling.  It doesn't handle multi-rune or large-width sequences
// properly in the expected string currently.
func DisplayLike(width, height int, text string) types.GomegaMatcher {
	expected := tcell.NewSimulationScreen("")
	expected.Init()
	expected.SetSize(width, height)

	row := -1
	col := -1
	for _, rn := range text {
		col++
		if col % width == 0 {
			row++
			col = 0
		}
		expected.SetContent(col, row, rn, nil, tcell.StyleDefault)
	}

	expected.Show()

	return &cellsMatcher{expected: expected, contentsOnly: true}
}

func DisplayWithStyle(width, height int, pairs ...interface{}) types.GomegaMatcher {
	if len(pairs) % 2 != 0 {
		panic("DisplayWithStyle expects pairs of (text, style)")
	}

	spans := make([]struct{txt string; sty tcell.Style}, len(pairs)/2)

	for i, item := range pairs {
		switch i % 2 {
		case 0:
			str, ok := item.(string)
			if !ok {
				panic("DisplayWithStyle expects pairs of (text, style)")
			}
			spans[i/2].txt = str
		case 1:
			sty, ok := item.(tcell.Style)
			if !ok {
				panic("DisplayWithStyle expects pairs of (text, style)")
			}
			spans[i/2].sty = sty
		}
	}

	expected := tcell.NewSimulationScreen("")
	expected.Init()
	expected.SetSize(width, height)

	row := -1
	col := -1
	for _, span := range spans {
		for _, rn := range span.txt {
			col++
			if col % width == 0 {
				row++
				col = 0
			}
			expected.SetContent(col, row, rn, nil, span.sty)
		}
	}

	expected.Show()

	return &cellsMatcher{expected: expected}
}

func DisplayWithCells(width, height int, cells ...tcell.SimCell) types.GomegaMatcher {
	expected := tcell.NewSimulationScreen("")
	expected.Init()
	expected.SetSize(width, height)

	row := -1
	col := -1
	for _, cell := range cells {
		col++
		if col % width == 0 {
			row++
			col = 0
		}
		expected.SetContent(col, row, cell.Runes[0], cell.Runes[1:], cell.Style)
	}

	expected.Show()

	return &cellsMatcher{expected: expected}
}