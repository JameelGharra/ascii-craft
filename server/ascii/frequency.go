package ascii

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JameelGharra/ascii-craft/server/utils"
)

type FreqTable struct {
	Entries             []*FreqEntry
	TotalDifferentChars int
	entriesCache        map[int]*FreqEntry // to avoid finding point on need
}
type FreqEntry struct {
	Value int
	Count int
}

func NewFrequency() *FreqTable {
	return &FreqTable{
		Entries:             make([]*FreqEntry, 0),
		entriesCache:        make(map[int]*FreqEntry),
		TotalDifferentChars: 0,
	}
}

func (f *FreqTable) Reset() {
	f.TotalDifferentChars = 0
	f.Entries = make([]*FreqEntry, 0)
	f.entriesCache = map[int]*FreqEntry{}
}

func (f *FreqTable) Count(dataIterator utils.IByteIterator) {
	for dataIterator.HasNext() {
		b := dataIterator.Next()
		entry, exists := f.entriesCache[int(b)]
		if !exists {
			entry = &FreqEntry{
				Value: b,
				Count: 0,
			}
			f.entriesCache[int(b)] = entry
			f.Entries = append(f.Entries, entry)
			f.TotalDifferentChars++
		}
		entry.Count++
	}
}

func (f *FreqTable) Debug() string {
	points := make([]*FreqEntry, len(f.Entries))
	copy(points, f.Entries)
	slices.SortFunc(points, func(a, b *FreqEntry) int {
		return a.Count - b.Count
	})

	var out strings.Builder
	out.WriteString(fmt.Sprintf("Frequency(%d): ", len(points)))
	top := 0
	bottom := 0
	for i, p := range points {
		if i+4 >= len(points) {
			top += p.Count
		} else {
			bottom += p.Count
		}
		fmt.Fprintf(&out, "%s(%d) ", string(rune(p.Value)), p.Count)
	}

	out.WriteString(fmt.Sprintf("-- top %d bottom %d diff %d", top, bottom, top-bottom))

	return out.String()
}
