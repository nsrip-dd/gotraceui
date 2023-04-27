package theme

import (
	"image"
	"image/color"
	"sort"

	"gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/exp/slices"
	"honnef.co/go/gotraceui/layout"
	"honnef.co/go/gotraceui/widget"
)

// TODO(dh): split FlameGraph into widget state, theme state and style

type FlameGraph struct {
	Color   func(level, idx int, f Frame) color.NRGBA
	samples []internalFrame
}

type Sample []Frame

type Frame struct {
	Name string
}

type internalFrame struct {
	Frame
	NumSamples int
	Children   []internalFrame
}

func (fg *FlameGraph) AddSample(sample Sample) {
	if len(sample) == 0 {
		return
	}

	toplevel := internalFrame{
		Frame: Frame{
			Name: "",
		},
		NumSamples: 1,
		Children: []internalFrame{
			{
				Frame:      sample[0],
				NumSamples: 1,
			},
		},
	}

	cur := &toplevel.Children[0]
	for i := range sample[1:] {
		child := internalFrame{
			Frame:      sample[i+1],
			NumSamples: 1,
		}
		cur.Children = append(cur.Children, child)
		cur = &cur.Children[0]
	}

	fg.samples = append(fg.samples, toplevel)
}

func (fg *FlameGraph) Compute() {
	var merge func(root []internalFrame) []internalFrame

	merge = func(slice []internalFrame) []internalFrame {
		if len(slice) == 0 {
			return nil
		}

		sort.Slice(slice, func(i, j int) bool {
			return slice[i].Name < slice[j].Name
		})
		for i := range slice[:len(slice)-1] {
			frame := &slice[i]
			if frame.NumSamples == 0 {
				continue
			}

			for j := i + 1; j < len(slice); j++ {
				next := &slice[j]
				if frame.Name != next.Name {
					break
				}
				frame.Children = append(frame.Children, next.Children...)
				frame.NumSamples += next.NumSamples
				next.NumSamples = 0
			}
		}

		compacted := slice[:0]
		for i := range slice {
			if slice[i].NumSamples > 0 {
				compacted = append(compacted, slice[i])
			}
		}
		slice = compacted

		for i := range slice {
			child := &slice[i]
			child.Children = merge(child.Children)
		}

		return slice
	}

	fg.samples = merge(fg.samples)

	if len(fg.samples) > 1 {
		panic("too many top-level samples")
	}
}

func (fg *FlameGraph) Layout(win *Window, gtx layout.Context) layout.Dimensions {
	// XXX figure out decent height. at least be high enough for the chosen font
	const height = 30
	const rowSpacing = 1
	const rowPadding = 2
	const maxRadius = 4

	// XXX handle graphs with no samples

	pxPerSample := float64(gtx.Constraints.Min.X) / float64(fg.samples[0].NumSamples)

	var do func(level int, startX int, samples []internalFrame)

	colorFn := fg.Color
	if colorFn == nil {
		colorFn = func(level, idx int, f Frame) color.NRGBA {
			return rgba(0xFF00FFFF)
		}
	}

	// Indices tracks the intra-row span index per level. This is useful for color functions that want to discern
	// neighboring spans.
	var indices []int
	do = func(level int, startX int, samples []internalFrame) {
		if len(indices) < level+1 {
			indices = slices.Grow(indices, level+1-len(indices))[:level+1]
		}

		x := startX
		for _, frame := range samples {
			width := int(float64(frame.NumSamples) * pxPerSample)
			if width == 0 {
				continue
			}

			idx := &indices[level]
			*idx++

			radius := maxRadius
			if maxRadius > width {
				radius = width
			}

			func() {
				y := gtx.Constraints.Min.Y - (height+gtx.Dp(rowSpacing))*(level+1)
				defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
				shape := clip.UniformRRect(image.Rectangle{Max: image.Pt(width, height)}, radius)
				c := colorFn(level, *idx, frame.Frame)
				paint.FillShape(gtx.Ops, c, shape.Op(gtx.Ops))

				gtx := gtx
				gtx.Constraints.Max.X = width - 2*gtx.Dp(rowPadding)
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				defer op.Offset(image.Pt(gtx.Dp(rowPadding), 0)).Push(gtx.Ops).Pop()
				widget.Label{MaxLines: 1, Alignment: text.Middle, HideIfEntirelyTruncated: true}.Layout(gtx, win.Theme.Shaper, font.Font{}, 12, frame.Name, widget.ColorTextMaterial(gtx, rgba(0x000000FF)))
			}()

			do(level+1, x, frame.Children)
			x += width

		}
	}

	do(0, 0, fg.samples)

	return layout.Dimensions{Size: gtx.Constraints.Min}
}
