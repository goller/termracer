package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// File is the name of a file to type.
type File string

// Line is the line number within the file.
type Line int

// Paragraph is a paragaph to type within the corpus.
type Paragraph struct {
	File     File `json:"file"`
	Line     Line `json:"line"`
	Finished bool `json:"finished"`
}

// Corpus is a record of paragraphs that have been typed.
type Corpus struct {
	Paragraphs []Paragraph `json:"paragraphs"`
}

// NewCorpus loads records of all things typed in the records file.
// Corpus will serve more paragraphs through the entire sample directory.
func NewCorpus(records string) (*Corpus, error) {
	if _, err := os.Stat(records); os.IsNotExist(err) {
		return &Corpus{}, nil
	}
	b, err := ioutil.ReadFile(records)
	if err != nil {
		return nil, err
	}
	var c Corpus
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Corpus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, err := c.chooseParagraph()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = ioutil.WriteFile("samples/record.json", b, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(w, string(p))
}

func (c *Corpus) chooseParagraph() ([]byte, error) {
	files, err := ioutil.ReadDir("samples/use")
	n := len(files)
	if err != nil || n == 0 {
		return nil, errors.New("failed to read use directory")
	}

	count := -1
BEGIN:
	count++
	if count == len(files) {
		return nil, fmt.Errorf("finished all files")
	}

	file := files[count]

	found := -1
	for i := range c.Paragraphs {
		if c.Paragraphs[i].File == File(file.Name()) {
			if c.Paragraphs[i].Finished {
				goto BEGIN
			}

			found = i
			c.Paragraphs[i].Line += Line(15)
			break
		}
	}

	if found < 0 {
		c.Paragraphs = append(c.Paragraphs, Paragraph{
			File: File(file.Name()),
		})

		found = len(c.Paragraphs) - 1
	}

	b, err := ioutil.ReadFile("samples/use/" + string(c.Paragraphs[found].File))
	if err != nil {
		return nil, errors.New("error in reading a paragraph file")
	}

	lines := bytes.Split(b, []byte("\n"))
	if len(lines) < 15 {
		c.Paragraphs[found].Finished = true
		return b, nil
	}

	start := int(c.Paragraphs[found].Line)

	// we've seen all lines
	if start > len(lines) {
		c.Paragraphs[found].Finished = true
		goto BEGIN
	}

	incr := len(lines[start:])
	if incr > 15 {
		incr = 15
	}

	end := start + incr

	paragraph := bytes.Join(lines[start:end], []byte("\n"))
	return paragraph, nil
}

func main() {
	h, err := NewCorpus("samples/record.json")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/paragraph", h)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
