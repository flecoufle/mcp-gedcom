package gedcom

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/iand/gedcom"
)

type ReverseAssoc struct {
	SourceXref string
	Relation   string
}

type Loader struct {
	mu              sync.RWMutex
	document        *gedcom.Gedcom
	path            string
	individualsMap  map[string]*gedcom.IndividualRecord
	familiesMap     map[string]*gedcom.FamilyRecord
	reverseAssocMap map[string][]ReverseAssoc
}

var instance atomic.Pointer[Loader]

func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	reader, err := GetGedcomReader(l.path)
	if err != nil {
		return fmt.Errorf("failed to read GEDCOM file: %w", err)
	}

	decoder := gedcom.NewDecoder(reader)
	doc, err := decoder.Decode()
	if err != nil {
		return fmt.Errorf("failed to parse GEDCOM file: %w", err)
	}

	l.document = doc

	// Build O(1) lookup maps
	l.individualsMap = make(map[string]*gedcom.IndividualRecord)
	for _, ind := range doc.Individual {
		l.individualsMap[ind.Xref] = ind
	}

	l.familiesMap = make(map[string]*gedcom.FamilyRecord)
	for _, fam := range doc.Family {
		l.familiesMap[fam.Xref] = fam
	}

	l.reverseAssocMap = make(map[string][]ReverseAssoc)
	for _, ind := range doc.Individual {
		for _, a := range ind.Association {
			if a.Xref == "" {
				continue
			}
			rel := strings.ToLower(a.Relation)
			if rel != "godparent" && rel != "godfather" && rel != "godmother" && rel != "godf" && rel != "godm" {
				continue
			}
			l.reverseAssocMap[a.Xref] = append(
				l.reverseAssocMap[a.Xref],
				ReverseAssoc{SourceXref: ind.Xref, Relation: a.Relation},
			)
		}
	}

	return nil
}

func (l *Loader) GetDocument() *gedcom.Gedcom {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.document
}

func (l *Loader) Individuals() []*gedcom.IndividualRecord {
	doc := l.GetDocument()
	if doc == nil {
		return nil
	}
	return doc.Individual
}

func (l *Loader) Individual(id string) *gedcom.IndividualRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.individualsMap[id]
}

func (l *Loader) Families() []*gedcom.FamilyRecord {
	doc := l.GetDocument()
	if doc == nil {
		return nil
	}
	return doc.Family
}

func (l *Loader) Family(id string) *gedcom.FamilyRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.familiesMap[id]
}

func (l *Loader) GetReverseAssociations(xref string) []ReverseAssoc {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.reverseAssocMap[xref]
}

func Get() *Loader {
	return instance.Load()
}

func Init(path string) error {
	l := NewLoader(path)
	if err := l.Load(); err != nil {
		return err
	}
	instance.Store(l)
	return nil
}

func ReloadFile(path string) error {
	l := NewLoader(path)
	if err := l.Load(); err != nil {
		return fmt.Errorf("failed to load GEDCOM file: %w", err)
	}
	instance.Store(l)
	return nil
}
