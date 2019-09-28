package main

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/jan25/gocui"
)

// Paragraph encapsulates the data and
// state of Paragraph view
type Paragraph struct {
	// name of View
	name string
	// positions, dimensions
	x, y int
	w, h int

	paragraph string

	// done channel
	done chan struct{}
	// words in paragraph
	words []span
	// index of current word being typed
	wordi int
	// whether current word is mistyped
	Mistyped bool
}

func newParagraph(name string, x, y int, w, h int) *Paragraph {
	// split into words at whitespace characters
	// words := strings.Fields(paragraph)

	// view.Wrap = true
	return &Paragraph{
		name: name,
		x:    x,
		y:    y,
		w:    w,
		h:    h,
	}
}

// Layout manager for paragraph View
func (p *Paragraph) Layout(g *gocui.Gui) error {
	v, err := g.SetView(p.name, p.x, p.y, p.x+p.w, p.y+p.h)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	v.Wrap = true

	select {
	case <-p.getDoneCh():
		// channel closed
		v.Clear()
	default:
		p.DrawView(v)
	}

	return nil
}

// Init initialises new paragraph to type
func (p *Paragraph) Init() {
	p.done = make(chan struct{})

	var err error
	p.paragraph, err = ChooseParagraph()
	if err != nil {
		Logger.Error(fmt.Sprintf("%v", err))
		return
	}

	p.words = fieldsFunc(p.paragraph, unicode.IsSpace)
	p.wordi = 0
}

// Advance moves target word to next word
func (p *Paragraph) Advance() error {
	if p.wordi >= len(p.words)-1 {
		return errors.New("can not advance beyond number of words")
	}

	p.wordi++
	return nil
}

// CountDoneWords returns number of words
// already types in current race
func (p *Paragraph) CountDoneWords() int {
	return p.wordi
}

// CurrentWord returns target word to type
func (p *Paragraph) CurrentWord() string {
	span := p.words[p.wordi]
	return p.paragraph[span.start:span.end]
}

// CharsUptoCurrent counts chars in words
// upto/including current word
func (p *Paragraph) CharsUptoCurrent() int {
	c := 0
	for i := 0; i < p.wordi; i++ {
		span := p.words[i]
		c += span.end - span.start
	}
	return c
}

// DrawView renders the paragraph View
func (p *Paragraph) DrawView(v *gocui.View) {
	v.Clear()
	if len(p.words) == 0 {
		return
	}

	colorized := "\033[32;7m%s\033[0m"
	if p.Mistyped {
		// Red/pink bg
		colorized = "\033[31;7m%s\033[0m"
	}
	span := p.words[p.wordi]
	colorized = fmt.Sprintf(colorized, p.paragraph[span.start:span.end])

	fmt.Fprintf(v, "%s%s%s", p.paragraph[:span.start], colorized, p.paragraph[span.end:])
}

func (p *Paragraph) getDoneCh() chan struct{} {
	if p.done == nil {
		p.done = make(chan struct{})
	}
	return p.done
}

// Reset deactivates the paragraph view
// used to stop a race
func (p *Paragraph) Reset() {
	select {
	case <-p.getDoneCh():
		// already closed
		// nothing to do
	default:
		close(p.getDoneCh())
	}
}

// A span is used to record a slice of s of the form s[start:end].
// The start index is inclusive and the end index is exclusive.
type span struct {
	start int
	end   int
}

func fieldsFunc(s string, f func(rune) bool) []span {
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	wasField := false
	fromIndex := 0

	for i, rune := range s {
		if f(rune) {
			if wasField {
				spans = append(spans, span{start: fromIndex, end: i})
				wasField = false
			}
		} else {
			if !wasField {
				fromIndex = i
				wasField = true
			}
		}
	}

	// Last field might end at EOF.
	if wasField {
		spans = append(spans, span{fromIndex, len(s)})
	}

	return spans
}
