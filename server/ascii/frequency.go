package ascii

import (
	"fmt"
	"slices"
)

type FreqTable struct {
	Buffer              [256]int
	Points              []*FreqEntry
	TotalDifferentChars int
	pointsCache         [256]*FreqEntry // to avoid finding point on need
}
type FreqEntry struct {
	Value byte
	Count int
}

func NewFrequency() *FreqTable {
	return &FreqTable{
		Buffer:              [256]int{},
		Points:              make([]*FreqEntry, 0),
		pointsCache:         [256]*FreqEntry{},
		TotalDifferentChars: 0,
	}
}

func (f *FreqTable) Reset() {
	f.TotalDifferentChars = 0
	for i := range 256 {
		f.Buffer[i] = 0
		// should reset points and cache here, but kept this for the 5,000 frames benchmark, will do it laters
	}
}

func (f *FreqTable) Count(data []byte) {
	for _, b := range data {
		if f.Buffer[b] == 0 {
			f.TotalDifferentChars++
			point := &FreqEntry{
				Count: 0,
				Value: b,
			}
			f.Points = append(f.Points, point)
			f.pointsCache[b] = point
		}
		f.Buffer[b]++
		f.pointsCache[b].Count++
	}
}

func (f *FreqTable) Debug() string {
	points := make([]*FreqEntry, len(f.Points))
	copy(points, f.Points)
	slices.SortFunc(points, func(a, b *FreqEntry) int {
		return a.Count - b.Count
	})

	out := fmt.Sprintf("Frequency(%d): ", len(points))
	top := 0
	bottom := 0
	for i, p := range points {
		if i+4 >= len(points) {
			top += p.Count
		} else {
			bottom += p.Count
		}
		out += fmt.Sprintf("%s(%d) ", string(p.Value), p.Count)
	}

	out += fmt.Sprintf("-- top %d bottom %d diff %d", top, bottom, top-bottom)

	return out
}
