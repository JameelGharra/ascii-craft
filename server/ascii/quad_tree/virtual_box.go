package quad_tree

import "github.com/JameelGharra/ascii-craft/server/utils"

type VirtualBox struct {
	buffer    []byte
	TotalRows int
	TotalCols int
	// per box
	BoxRows  int
	BoxCols  int
	RowStart int
	ColStart int

	Stride int // for data consumption

	iterRow int
	iterCol int
}

func NewVirtualBox(buffer []byte, totalRows, totalCols int) *VirtualBox {
	return &VirtualBox{
		buffer:    buffer,
		RowStart:  0,
		ColStart:  0,
		TotalRows: totalRows,
		TotalCols: totalCols,
		BoxRows:   totalRows,
		BoxCols:   totalCols,
		iterRow:   0,
		iterCol:   0,
	}
}

func fromVirtualBox(vb *VirtualBox) *VirtualBox {
	return &VirtualBox{
		buffer:    vb.buffer,
		RowStart:  0,
		ColStart:  0,
		TotalRows: vb.TotalRows,
		TotalCols: vb.TotalCols,
		Stride:    vb.Stride,
		iterRow:   0,
		iterCol:   0,
	}
}

func (vb *VirtualBox) withSize(boxRows, boxCols int) *VirtualBox {
	vb.BoxRows = boxRows
	vb.BoxCols = boxCols
	return vb
}

func (vb *VirtualBox) Len() int {
	return vb.BoxRows * vb.BoxCols
}

func (vb *VirtualBox) setStart(rowStart, colStart int) *VirtualBox {
	vb.RowStart = rowStart
	vb.ColStart = colStart
	return vb
}

func (vb *VirtualBox) withStride(stride int) *VirtualBox {
	utils.Assert(stride == 1 || stride == 2, "virtual boxes supported are 8 or 16 bit values")
	vb.Stride = stride
	return vb
}

func (vb *VirtualBox) Next() (isDone bool, value int) {
	utils.Assert(vb.iterRow == vb.BoxRows, "box iterator already finished")
	row := vb.RowStart + vb.iterRow
	col := vb.ColStart + vb.iterCol
	index := TranslateTo1D(row, col, vb.TotalCols)

	utils.Assert(index >= 0 && index < len(vb.buffer), "exceeded indices while iterating over a virtual box", "idx", index, "row", row, "col", col)
	if vb.Stride == 2 {
		value = utils.Read16(vb.buffer, index)
	} else {
		value = int(vb.buffer[index])
	}
	vb.iterCol += vb.Stride
	if vb.iterCol == vb.BoxCols {
		vb.iterCol = 0
		vb.iterRow++
	}
	isDone = vb.iterRow == vb.BoxRows
	return isDone, value
}

func (vb *VirtualBox) ResetIterator() {
	vb.iterRow = 0
	vb.iterCol = 0
}

// would return boxes on order of top left, top right, bottom left, bottom right
func (vb *VirtualBox) quad() (*VirtualBox, *VirtualBox, *VirtualBox, *VirtualBox) {
	utils.Assert(vb.Stride > 0, "stride is not set for box to quad")
	utils.Assert(vb.BoxRows >= 2 && vb.BoxCols >= 2, "virtual box too small to quad, has to be at least 2x2")

	newBoxEachRows := vb.BoxRows / 2
	newBoxEachCols := vb.BoxCols / 2
	newBoxEachCols -= newBoxEachCols % vb.Stride // get it back 1 pixel for alignment

	centerNewRowIndex := vb.RowStart + newBoxEachRows
	centerNewColIndex := vb.ColStart + newBoxEachCols

	maxColIndex := centerNewColIndex + (vb.BoxCols - newBoxEachCols) - 1
	maxRowIndex := centerNewRowIndex + (vb.BoxRows - newBoxEachRows) - 1

	maxIndex := TranslateTo1D(maxRowIndex, maxColIndex, vb.TotalCols)
	utils.Assert(maxIndex < len(vb.buffer), "virtual box exceeds buffer size on translation for quad")

	topLeft := fromVirtualBox(vb).
		setStart(vb.RowStart, vb.ColStart).
		withSize(newBoxEachRows, newBoxEachCols)

	topRight := fromVirtualBox(vb).
		setStart(vb.RowStart, centerNewColIndex).
		withSize(vb.BoxRows, vb.BoxCols-newBoxEachCols)

	bottomLeft := fromVirtualBox(vb).
		setStart(centerNewRowIndex, vb.ColStart).
		withSize(vb.BoxRows-newBoxEachRows, newBoxEachCols)

	bottomRight := fromVirtualBox(vb).
		setStart(centerNewRowIndex, centerNewColIndex).
		withSize(vb.BoxRows-newBoxEachRows, vb.BoxCols-newBoxEachCols)

	return topLeft, topRight, bottomLeft, bottomRight
}

func partition(data []byte, currentVb *VirtualBox, boxes *[]*VirtualBox, depth int) {
	if depth == 0 {
		*boxes = append(*boxes, currentVb)
	}
	tl, tr, bl, br := currentVb.quad()
	partition(data, tl, boxes, depth-1)
	partition(data, tr, boxes, depth-1)
	partition(data, bl, boxes, depth-1)
	partition(data, br, boxes, depth-1)
}

func Partition(data []byte, rows, cols, depth, stride int) []*VirtualBox {
	boxes := &([]*VirtualBox{})
	vb := NewVirtualBox(data, rows, cols).
		withStride(stride)
	partition(data, vb, boxes, depth)
	return *boxes
}
