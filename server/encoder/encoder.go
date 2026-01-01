package encoder

import (
	"github.com/JameelGharra/ascii-craft/server/ascii/quad_tree"
)

type EncodingFrame struct {
	Prev []byte
	Curr []byte

	CurrQT quad_tree.QuadTree
	PrevQT quad_tree.QuadTree

	Out []byte
	Len int
}

func NewEncodingFrame(size int, params quad_tree.QuadTreeParam) *EncodingFrame {
	out := make([]byte, size)
	prevQt := quad_tree.Partition(out, params)
	currQt := quad_tree.Partition(out, params)
	return &EncodingFrame{
		Prev:   nil,
		PrevQT: prevQt,

		Curr:   nil,
		CurrQT: currQt,

		Out: out,
		Len: 0,
	}
}

func (ef *EncodingFrame) pushFrame(frame []byte) {
	ef.Prev = ef.Curr
	ef.Curr = frame

	ef.CurrQT.UpdateBuffer(ef.Curr)
	if ef.Prev != nil {
		ef.PrevQT.UpdateBuffer(ef.Prev)
	}
}

type EncodingCall func(frame *EncodingFrame) error

type Encoder struct {
	encodings      []EncodingCall
	frames         []*EncodingFrame
	size           int
	quadTreeParams quad_tree.QuadTreeParam
}

func NewEncoder(size int, treeParams quad_tree.QuadTreeParam) *Encoder {
	return &Encoder{
		encodings:      make([]EncodingCall, 0),
		frames:         make([]*EncodingFrame, 0),
		size:           size,
		quadTreeParams: treeParams,
	}
}

func (e *Encoder) AddEncoding(encoding EncodingCall) {
	e.encodings = append(e.encodings, encoding)
	e.frames = append(e.frames, NewEncodingFrame(e.size, e.quadTreeParams))
}

func (e *Encoder) PushFrame(rawData []byte) (int, []byte) {
	minSizeTarget := len(rawData)
	minBytes := rawData
	for i, encoding := range e.encodings {
		frame := e.frames[i]
		frame.pushFrame(rawData)
		err := encoding(frame)
		if err != nil {
			continue
		}
		if frame.Len < minSizeTarget {
			minSizeTarget = frame.Len
			minBytes = frame.Out
		}
	}
	return minSizeTarget, minBytes
}
