package stodupremover

import (
	"log"
	"os"
)

type Item struct {
	Filename string
}

type Actioner interface {
	Duplicate(Item, string) error
	NotDuplicate(Item) error
	Finish() error
}

type LoggerActioner struct {
	duplicates    int
	notDuplicates int
}

func (l *LoggerActioner) Duplicate(item Item, duplicateFilename string) error {
	l.duplicates++

	log.Printf("DUP %s (%s)", item.Filename, duplicateFilename)

	return nil
}

func (l *LoggerActioner) NotDuplicate(item Item) error {
	l.notDuplicates++

	log.Printf("NOT %s", item.Filename)

	return nil
}

func (l *LoggerActioner) Finish() error {
	log.Printf(
		"Finished; duplicates=%d notDuplicates=%d",
		l.duplicates,
		l.notDuplicates)

	return nil
}

type RemoveDuplicatesActioner struct{}

func (l *RemoveDuplicatesActioner) Duplicate(item Item, _ string) error {
	return os.Remove(item.Filename)
}

func (l *RemoveDuplicatesActioner) NotDuplicate(item Item) error {
	return nil
}

func (l *RemoveDuplicatesActioner) Finish() error {
	return nil
}

// can be used to, fox example, log AND remove duplicates
type TeeActioner struct {
	a Actioner
	b Actioner
}

func (t *TeeActioner) Duplicate(item Item, duplicateFilename string) error {
	if err := t.a.Duplicate(item, duplicateFilename); err != nil {
		return err
	}

	return t.b.Duplicate(item, duplicateFilename)
}

func (t *TeeActioner) NotDuplicate(item Item) error {
	if err := t.a.NotDuplicate(item); err != nil {
		return err
	}

	return t.b.NotDuplicate(item)
}

func (t *TeeActioner) Finish() error {
	if err := t.a.Finish(); err != nil {
		return err
	}

	return t.b.Finish()
}
