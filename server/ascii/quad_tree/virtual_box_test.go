package quad_tree

import (
	"testing"
)

func TestVirtualBoxConstructionNoAssert(t *testing.T) {
	rows := 50
	cols := 160
	size := cols * rows
	data := make([]byte, size)
	for i := range size {
		data[i] = byte(i % 256)
	}
	params := QuadTreeParam{
		Depth:  2,
		Rows:   rows,
		Cols:   cols,
		Stride: 1,
	}
	_ = Partition(data, params)
	params.Stride = 2
	_ = Partition(data, params)
}

func returnDataSetWithResult() (data, result []byte) {
	data = []byte{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}
	result = []byte{
		1, 2, 5, 6,
		3, 4, 7, 8,
		9, 10, 13, 14,
		11, 12, 15, 16,
	}
	return
}

func TestVirtualBoxConstructionOutput(t *testing.T) {
	data, expectedOutOrder := returnDataSetWithResult()
	params := QuadTreeParam{
		Depth:  2,
		Rows:   4,
		Cols:   4,
		Stride: 1,
	}
	boxes := Partition(data, params)
	if len(boxes) != 16 {
		t.Fatalf("Expected 16 boxes from partitioning 4x4 with depth 2, got %d", len(boxes))
	}
	for i, b := range boxes {
		idx := TranslateTo1D(b.RowStart, b.ColStart, b.TotalCols)
		// would be the same if I did a geometry check like this:
		// if i+1 != expectedOutOrder[idx]
		if int(data[idx]) != int(expectedOutOrder[i]) {
			t.Fatalf("Expected box %d to start with value %d, got %d", i, int(expectedOutOrder[i]), int(data[idx]))
		}
	}
}

// this one uses stride = 1 though
func TestVirtualBoxIterators(t *testing.T) {
	data, _ := returnDataSetWithResult()
	expectedResult := [][]int{
		{1, 2, 5, 6},
		{3, 4, 7, 8},
		{9, 10, 13, 14},
		{11, 12, 15, 16},
	}
	depth := 1
	params := QuadTreeParam{
		Depth:  depth,
		Rows:   4,
		Cols:   4,
		Stride: 1,
	}
	boxes := Partition(data, params)
	if len(boxes) != 4*depth {
		t.Fatalf("Expected 4 boxes from partitioning 4x4 with depth %d, got %d", depth, len(boxes))
	}
	var value int
	for i, b := range boxes {
		count := 0
		expectedBox := expectedResult[i]
		for expectedValue := range expectedBox {
			value = b.Next()
			if !b.HasNext() && count < len(expectedBox)-1 {
				t.Fatalf("Box %d finished early at value index %d", i, expectedValue)
			}
			if value != expectedBox[expectedValue] {
				t.Fatalf("Box %d at value index %d: expected %d, got %d", i, expectedValue, expectedBox[expectedValue], value)
			}
			count++
		}
		count = 0
		if b.HasNext() {
			t.Fatal("Expected box", i, "to be done after full iteration")
		}
	}
}

func TestVirtualBoxStride2(t *testing.T) {
	data, _ := returnDataSetWithResult()
	expectedResult := [][]int{
		{0x0102, 0x506},
		{0x0304, 0x0708},
		{0x090a, 0x0d0e},
		{0x0b0c, 0x0f10},
	}
	depth := 1
	params := QuadTreeParam{
		Depth:  depth,
		Rows:   4,
		Cols:   4,
		Stride: 2,
	}
	boxes := Partition(data, params)
	if len(boxes) != 4*depth {
		t.Fatalf("Expected 4 boxes from partitioning 4x4 with depth %d, got %d", depth, len(boxes))
	}
	var value int
	for i, b := range boxes {
		count := 0
		expectedBox := expectedResult[i]
		for expectedValue := range expectedBox {
			value = b.Next()
			if !b.HasNext() && count < len(expectedBox)-1 {
				t.Fatalf("Box %d finished early at value index %d", i, expectedValue)
			}
			if value != expectedBox[expectedValue] {
				t.Fatalf("Box %d at value index %d: expected %d, got %d", i, expectedValue, expectedBox[expectedValue], value)
			}
			count++
		}
		count = 0
		if b.HasNext() {
			t.Fatal("Expected box", i, "to be done after full iteration")
		}
	}
}
