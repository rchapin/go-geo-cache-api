package geostore

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

const outputPath = "/var/tmp/geocache-api-map.png"

type Stack[T any] struct {
	stack []*T
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.stack) == 0
}

func (s *Stack[T]) Push(q *T) {
	// Add the new value to the end of the slice, which acts as the "top" of the Stack
	s.stack = append(s.stack, q)
}

func (s *Stack[T]) Pop() *T {
	if s.IsEmpty() {
		return nil
	} else {
		// Get the index of the "top" of the Stack
		idx := len(s.stack) - 1
		// Retrieve the element from the stack that we are going to "pop" and return
		retval := s.stack[idx]
		// Remove it from the underlying slice by re-slicing the slice.
		s.stack = s.stack[:idx]
		return retval
	}
}

func printMap(g GeoStore, scale int) {
	// In order to build a visual representation of the map we will use a 2-dimensional array
	// (slices actually). Because grid coordinates start at 0 and go to N, we have a bit of an
	// "off-by-one" issue and for the math to work out cleanly we will allocate a grid that is x+1
	// and y+1 in size.
	// Get the size of the Root map in units and allocate our grid
	rootQT := g.getRootQuadTree()
	rows := int(rootQT.Quadrant.YMax) * scale
	cols := int(rootQT.Quadrant.XMax) * scale
	grid := make([][]byte, rows+1)
	for i := range grid {
		grid[i] = make([]byte, cols+1)
	}

	// Now that we have a grid into which we will write a generic representation of the map that we
	// can then use to output to various formats, we will execute a post-order, DFS traversal of the
	// map's *QuadTrees
	var current *QuadTree
	visited := make(map[uint64]bool)
	stack := &Stack[QuadTree]{}
	stack.Push(rootQT)
	for !stack.IsEmpty() {
		current = stack.Pop()
		_, ok := visited[current.id]
		if ok {
			// We have already processed this node and all of its children
			continue
		}

		if current.isSubdivided {
			// Add all of the children to the stack and continue the loop
			stack.Push(current.NW)
			stack.Push(current.NE)
			stack.Push(current.SW)
			stack.Push(current.SE)
			continue
		} else {
			// We are at the bottom of a given tree and can render it.  Pop the item off the stack,
			// render it and then continue.
			writeQuadTree(grid, rows, scale, current)
			visited[current.id] = true
		}
	}
	writeGridToPng(rows, cols, scale, grid)
}

func writeGridToPng(rows, cols, scale int, grid [][]byte) {
	// Ensure that the file does not already exist
	_, err := os.Stat(outputPath)
	if !os.IsNotExist(err) {
		err := os.Remove(outputPath)
		if err != nil {
			panic(err)
		}
	}

	topLeft := image.Point{0, 0}
	bottomRight := image.Point{cols, rows}
	img := image.NewRGBA(image.Rectangle{topLeft, bottomRight})
	white := color.RGBA{255, 255, 255, 0xff}
	black := color.RGBA{0, 0, 0, 0xff}
	grey := color.RGBA{196, 196, 196, 0xff}

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			switch b := grid[y][x]; {
			case b == byte(0):
				img.Set(x, y, white)
			case b == byte('.'):
				img.Set(x, y, grey)
			case b == byte('x'):
				img.Set(x, y, black)
			}
		}
	}

	// Encode the image as a PNG and write out the file
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	png.Encode(f, img)
	err = f.Close()
	if err != nil {
		panic(err)
	}
}

func writeHorzLine(y, xStart, xEnd int, grid [][]byte) {
	row := grid[y]
	for i := xStart; i <= xEnd; i++ {
		row[i] = byte('.')
	}
}

func writeVertLine(x, yStart, yEnd int, grid [][]byte) {
	for i := yStart; i > yEnd; i-- {
		grid[i][x] = byte('.')
	}
}

func writeQuadTree(grid [][]byte, rows, scale int, q *QuadTree) {
	// Because we are translating a grid coordinate system where 0,0 is the "bottom-left" to a 2-d
	// array where 0 in the top-level array equates to the nth, last, row in the grid, we need to
	// translate the address space for each of the points.
	quadrant := q.Quadrant
	xMin := gToAx(int(quadrant.XMin) * scale)
	xMax := gToAx(int(quadrant.XMax) * scale)
	yMin := gToAy(rows, int(quadrant.YMin)*scale)
	yMax := gToAy(rows, int(quadrant.YMax)*scale)
	writeHorzLine(yMax, xMin, xMax, grid)
	writeHorzLine(yMin, xMin, xMax, grid)
	writeVertLine(xMin, yMin, yMax, grid)
	writeVertLine(xMax, yMin, yMax, grid)

	// Write the points for each of the nodes
	for _, node := range q.Nodes {
		x := gToAx(int(node.X) * scale)
		y := gToAy(rows, int(node.Y)*scale)
		grid[y][x] = byte('x')
	}
}

// gToAx translates a grid coordinate to a 2-D array coordinate in the x axis
func gToAx(i int) int {
	if i < 1 {
		return 0
	} else {
		return i - 1
	}
}

// gToAy translates a grid coordinate to a 2-D array coordinate in the y axis
func gToAy(rows int, i int) int {
	retval := rows - i
	if retval < 1 {
		return 0
	}
	return retval
}
