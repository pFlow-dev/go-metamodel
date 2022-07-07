package image

import (
	"bufio"
	"fmt"
	"github.com/ajstarks/svgo"
	"github.com/pflow-dev/go-metamodel/metamodel"
	"io"
	"math"
	"os"
)

type svgImage struct {
	*svg.SVG
	stateMachine metamodel.Process
	width        int
	height       int
	writerOut    io.Writer
	onClose      func()
}

func NewSvgFile(outputPath string, xy ...int) *svgImage {
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	i := NewSvg(w, xy...)
	i.onClose = func() {
		err := w.Flush()
		if err != nil {
			panic(err)
		}
	}
	return i
}

func NewSvg(out io.Writer, xy ...int) *svgImage {
	i := new(svgImage)
	i.writerOut = out
	return i.newSvgImage(xy...)
}

/*
newSvgImage(w, h, minx, miny, vw, vh)
passes along parameters as viewbox
*/
func (i *svgImage) newSvgImage(xy ...int) *svgImage {
	i.SVG = svg.New(i.writerOut)
	if len(xy) == 2 {
		i.width = xy[0]
		i.height = xy[1]
	} else {
		i.width = 1024
		i.height = 768
	}

	if len(xy) == 6 {
		i.Startview(xy[0], xy[1], xy[2], xy[3], xy[4], xy[5])
	} else {
		i.Startview(i.width, i.height, 0, 0, i.width, i.height)
	}
	i.markerInhibit()
	i.markerArrow()
	return i
}

func (i *svgImage) markerArrow() {
	i.Marker("markerArrow1", 31, 6, 23, 13, `fill="#000000" stroke="#000000" orient="auto"`)
	i.Rect(3, 5, 28, 3, `fill="#ffffff" stroke="#ffffff"`) // cover end of lines
	i.Path("M2,2 L2,11 L10,6 L2,2")
	i.MarkerEnd()
}

func (i *svgImage) markerInhibit() {
	i.Marker("markerInhibit1", 31, 6, 23, 13, `fill="#000000" stroke="#000000" orient="auto"`)
	i.Rect(3, 5, 28, 3, `fill="#ffffff" stroke="#ffffff"`) // cover end of lines
	i.Circle(5, 6, 4)
	i.MarkerEnd()
}

func (i *svgImage) Render(net *metamodel.Model, initialVectors ...metamodel.Vector) {
	i.stateMachine = net.Execute(initialVectors...)
	for _, a := range net.Arcs {
		i.arc(a)
	}
	for _, p := range net.Places {
		i.place(p)
	}
	for _, t := range net.Transitions {
		i.transition(t)
	}
	i.End()
	if i.onClose != nil {
		i.onClose()
	}
}

func (i *svgImage) place(place *metamodel.Place) {
	i.Group()

	i.Circle(int(place.X), int(place.Y), 20, `strokeWidth="1.5" fill="#ffffff" stroke="#000000" orient="0" shapeRendering="auto"`)
	i.Text(int(place.X)-20, int(place.Y)-25, place.Label, `font-size="small"`)
	x := int(place.X)
	y := int(place.Y)

	tokens := i.stateMachine.TokenCount(place.Label)
	if tokens > 0 {
		if tokens == 1 {
			i.Circle(x, y, 2, `fill="#000000" stroke="#000000" orient="0" className="tokens"`)
		} else if tokens < 10 {
			i.Text(x-4, y+5, fmt.Sprintf("%v", tokens), `font-size="large"`)
		} else if tokens >= 10 {
			i.Text(x-7, y+5, fmt.Sprintf("%v", tokens), `font-size="small"`)
		}
	}
	i.Gend()
}

func (i *svgImage) arc(arc metamodel.Arc) {
	i.Group()

	var (
		y1     int64 = 0
		x1     int64 = 0
		y2     int64 = 0
		x2     int64 = 0
		weight int64 = 0
		marker       = "url(#markerArrow1)"
	)
	if arc.Inhibitor {
		marker = "url(#markerInhibit1)"
		if arc.Target.IsTransition() {
			p := arc.Source.GetPlace()
			t := arc.Target.GetTransition()
			g, ok := t.Guards[p.Label]
			if !ok {
				panic("missing guard: " + p.Label)
			}
			weight = g.Delta[p.Offset]
		} else {
			panic("invalid inhibitor")
		}
	} else {
		if arc.Source.IsTransition() {
			t := arc.Source.GetTransition()
			p := arc.Target.GetPlace()
			weight = t.Delta[p.Offset]
		} else if arc.Target.IsTransition() {
			p := arc.Source.GetPlace()
			t := arc.Target.GetTransition()
			weight = t.Delta[p.Offset]
		} else {
			panic("invalid arc")
		}
	}
	if arc.Source.IsPlace() {
		p := arc.Source.GetPlace()
		y1 = p.Y
		x1 = p.X
		t := arc.Target.GetTransition()
		y2 = t.Y
		x2 = t.X
	} else if arc.Source.IsTransition() {
		t := arc.Source.GetTransition()
		y1 = t.Y
		x1 = t.X
		p := arc.Target.GetPlace()
		y2 = p.Y
		x2 = p.X
	} else {
		panic("invalid arc declaration")
	}

	var midX int64 = (x2 + x1) / 2
	var midY int64 = (y2+y1)/2 - 8
	var offsetX int64 = 4
	var offsetY int64 = 4

	if math.Abs(float64(x2-midX)) < 8 {
		offsetX = 8
	}

	if math.Abs(float64(x2-midY)) < 8 {
		offsetY = 0
	}

	if weight < 0 {
		weight = 0 - weight
	}

	i.Line(int(x1), int(y1), int(x2), int(y2), `stroke="#000000" marker-end="`+marker+`"`)
	i.Text(int(midX-offsetX), int(midY+offsetY), fmt.Sprintf("%v", weight), `font-size="small"`)
	i.Gend()
}

func (i *svgImage) transition(transition *metamodel.Transition) {
	i.Group()

	op := metamodel.Op{Action: transition.Label, Multiple: 1, Role: transition.Role.Label}

	var fill = "#ffffff"
	{
		valid, _, _ := i.stateMachine.TestFire(op)
		inhibited, _ := i.stateMachine.Inhibited(op)

		if !valid && inhibited {
			fill = "#fab5b0"
		} else if valid {
			fill = "#62fa75"
		}
	}

	x := int(transition.X - 17)
	y := int(transition.Y - 17)
	i.Rect(x, y, 34, 34, `stroke="#000000" fill="`+fill+`"`)
	i.Text(x, y-8, transition.Label, `font-size="small"`)
	i.Gend()
}
