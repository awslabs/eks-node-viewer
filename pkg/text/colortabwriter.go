/*
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

package text

import (
	"io"
)

const debug = false

type ColorTabWriter struct {
	output   io.Writer
	padding  int
	minWidth int
	tabWidth int

	contents   [][][]byte
	cellWidths []int
}

func NewColorTabWriter(output io.Writer, minWidth, tabWidth, padding int) *ColorTabWriter {
	return &ColorTabWriter{
		output:   output,
		minWidth: minWidth,
		tabWidth: tabWidth,
		padding:  padding,
	}
}

func (c *ColorTabWriter) Write(buf []byte) (n int, err error) {
	for _, ch := range buf {
		switch ch {
		case '\t':
			c.newCell()
		case '\n':
			c.newLine()
		default:
			c.append(ch)
		}
	}
	return len(buf), nil
}

func (c *ColorTabWriter) Flush() {
	maxWidth := 0
	for _, line := range c.contents {
		// ensure we track a cell width for every cell
		for len(line) > len(c.cellWidths) {
			c.cellWidths = append(c.cellWidths, 0)
		}
		for i, cell := range line {
			cellLen := strlen(cell)
			if cellLen > c.cellWidths[i] {
				c.cellWidths[i] = cellLen
			}
			if cellLen > maxWidth {
				maxWidth = cellLen
			}
		}
	}

	padding := make([]byte, maxWidth+2)
	for i := range padding {
		if debug {
			padding[i] = '.'
		} else {
			padding[i] = ' '
		}
	}

	for _, line := range c.contents {
		if len(line) == 0 {
			continue
		}
		for i, cell := range line {
			// collapse empty columns
			if c.cellWidths[i] == 0 {
				continue
			}
			cellStrLen := strlen(cell)
			cellPadding := c.cellWidths[i] + c.padding - cellStrLen

			if debug {
				c.output.Write([]byte("|"))
			}
			c.output.Write(cell)
			c.output.Write(padding[0:cellPadding])
		}
		c.output.Write([]byte("\n"))
	}
	c.contents = nil
	c.cellWidths = nil
}

func strlen(cell []byte) int {
	nChars := 0
	inEscape := false
	for _, c := range cell {
		switch c {
		case 0x1b:
			inEscape = true
		case 'm':
			if inEscape {
				inEscape = false
			} else {
				nChars++
			}
		default:
			if !inEscape {
				nChars++
			}
		}
	}
	return nChars
}

func (c *ColorTabWriter) newCell() {
	if len(c.contents) == 0 {
		c.newLine()
	}
	c.contents[len(c.contents)-1] = append(c.contents[len(c.contents)-1], []byte{})
}

func (c *ColorTabWriter) newLine() {
	c.contents = append(c.contents, [][]byte{})
}

func (c *ColorTabWriter) append(ch byte) {
	if len(c.contents) == 0 {
		c.newLine()
	}
	lastLine := len(c.contents) - 1
	if len(c.contents[lastLine]) == 0 {
		c.newCell()
	}
	lastCell := len(c.contents[lastLine]) - 1
	c.contents[lastLine][lastCell] = append(c.contents[lastLine][lastCell], ch)
}
