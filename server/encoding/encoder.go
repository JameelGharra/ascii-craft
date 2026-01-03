package encoding

// import (
// 	"bytes"

// 	"github.com/JameelGharra/ascii-craft/server/ascii"
// 	"github.com/JameelGharra/ascii-craft/server/utils"
// )

// type FrameEncoder struct {
// 	buffers           []*ascii.AsciiFrame // for prev-current diffing
// 	rleExecutor       *AsciiRLE
// 	diff              *ascii.AsciiFrame
// 	TotalOriginalData int
// 	DeflatedTotalData int
// 	frameCount        int
// }

// func NewFrameEncoder(width, height uint32) *FrameEncoder {
// 	return &FrameEncoder{
// 		buffers: []*ascii.AsciiFrame{
// 			ascii.NewAsciiFrame(width, height),
// 			ascii.NewAsciiFrame(width, height),
// 		},
// 		rleExecutor:       NewAsciiRLE(),
// 		diff:              ascii.NewAsciiFrame(width, height),
// 		TotalOriginalData: 0,
// 		DeflatedTotalData: 0,
// 		frameCount:        0,
// 	}
// }

// func (fe *FrameEncoder) Encode(frame *ascii.AsciiFrame) ([]byte, error) {
// 	prev, curr := fe.buffers[fe.frameCount%2], fe.buffers[(fe.frameCount+1)%2]
// 	curr.Push(frame.Buffer)
// 	// *curr = *frame
// 	fe.diff.Xor(curr, prev)
// 	fe.rleExecutor.Reset()
// 	if err := fe.rleExecutor.Write(fe.diff.Buffer); err != nil {
// 		return nil, err
// 	}
// 	fe.rleExecutor.Finish()
// 	fe.TotalOriginalData += len(fe.diff.Buffer)
// 	fe.DeflatedTotalData += fe.rleExecutor.Size()
// 	fe.frameCount++

// 	result, err := fe.rleExecutor.Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	//
// 	decoded := fe.Decode(result, prev)
// 	utils.Assert(bytes.Equal(decoded.Buffer, curr.Buffer), "RLE Decode mismatch detected")
// 	//
// 	return result, nil
// }
// func (fe *FrameEncoder) Result() *ascii.AsciiFrame {
// 	return fe.buffers[fe.frameCount%2]
// }

// func (fe *FrameEncoder) Decode(data []byte, prev *ascii.AsciiFrame) *ascii.AsciiFrame {
// 	// Decoding RLE manually
// 	diffBuffer := make([]byte, prev.Width*prev.Height*4) // cant be more than x2 planared size
// 	pos := 0
// 	for i := 0; i < len(data); i += 2 {
// 		count := int(data[i])
// 		value := data[i+1]
// 		for j := 0; j < count; j++ {
// 			diffBuffer[pos] = value
// 			pos++
// 		}
// 	}
// 	diffFrame := ascii.NewAsciiFrame(prev.Width, prev.Height).Push(diffBuffer[:pos])
// 	return ascii.NewAsciiFrame(prev.Width, prev.Height).Xor(prev, diffFrame)
// }
