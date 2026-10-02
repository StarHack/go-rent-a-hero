package engine

import (
	"math"

	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

type Point struct {
	X, Y int
}

const NavCellSize = 4

const (
	navVisited = byte(0x01)
	navDynamic = byte(0x40)
	navBlocked = byte(0x80)
)

type NavGrid struct {
	raster     *zbf.Raster
	cellSize   int
	cols, rows int
	flags      []byte
}

func BuildNavGrid(raster *zbf.Raster) *NavGrid {
	cellSize := NavCellSize
	cols := raster.Width / cellSize
	rows := raster.Height / cellSize
	g := &NavGrid{
		raster:   raster,
		cellSize: cellSize,
		cols:     cols,
		rows:     rows,
		flags:    make([]byte, cols*rows),
	}

	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			x0 := cx * cellSize
			y0 := cy * cellSize
			blocked := false
			for y := y0; y < y0+cellSize && !blocked; y++ {
				for x := x0; x < x0+cellSize; x++ {
					if !raster.Walkable(x, y) {
						blocked = true
						break
					}
				}
			}
			if blocked {
				g.flags[g.index(cx, cy)] = navBlocked
			}
		}
	}

	for cx := 0; cx < cols; cx++ {
		g.flags[g.index(cx, 0)] = navBlocked
		g.flags[g.index(cx, rows-1)] = navBlocked
	}
	for cy := 0; cy < rows; cy++ {
		g.flags[g.index(0, cy)] = navBlocked
		g.flags[g.index(cols-1, cy)] = navBlocked
	}

	return g
}

func (g *NavGrid) index(cx, cy int) int {
	return cx*g.rows + cy
}

func (g *NavGrid) CellAt(x, y int) (int, int) {
	return x / g.cellSize, y / g.cellSize
}

func (g *NavGrid) CellCenter(cx, cy int) (int, int) {
	return cx*g.cellSize + 2, cy*g.cellSize + 2
}

func (g *NavGrid) SetDynamicRect(x1, y1, x2, y2 int, blocked bool) {
	if g == nil {
		return
	}
	cx1, cy1 := x1/g.cellSize, y1/g.cellSize
	cx2, cy2 := x2/g.cellSize, y2/g.cellSize
	if cx1 > cx2 {
		cx1, cx2 = cx2, cx1
	}
	if cy1 > cy2 {
		cy1, cy2 = cy2, cy1
	}
	for cx := cx1; cx <= cx2; cx++ {
		for cy := cy1; cy <= cy2; cy++ {
			if !g.inBounds(cx, cy) {
				continue
			}
			i := g.index(cx, cy)
			if blocked {
				g.flags[i] |= navDynamic
			} else {
				g.flags[i] &^= navDynamic
			}
		}
	}
}

func (g *NavGrid) inBounds(cx, cy int) bool {
	return cx >= 0 && cy >= 0 && cx < g.cols && cy < g.rows
}

func (g *NavGrid) cellFlags(cx, cy int) byte {
	if !g.inBounds(cx, cy) {
		return navBlocked
	}
	return g.flags[g.index(cx, cy)]
}

func (g *NavGrid) walkableCell(cx, cy int) bool {
	return g.cellFlags(cx, cy) == 0
}

func (g *NavGrid) clearVisited() {
	for i := range g.flags {
		g.flags[i] &^= navVisited
	}
}

var snapOffsets = [...]Point{
	{0, -1},
	{1, -1},
	{1, 0},
	{1, 1},
	{0, 1},
	{-1, 1},
	{-2, 0},
	{-1, -1},
}

func (g *NavGrid) adjustBlockedCell(cx, cy int) (Point, bool) {
	if g.cellFlags(cx, cy) == 0 {
		return Point{cx, cy}, true
	}
	for _, off := range snapOffsets {
		nx := cx + off.X
		ny := cy + off.Y
		if nx <= 0 || nx >= g.cols-1 || ny <= 0 || ny >= g.rows-1 {
			continue
		}
		if g.cellFlags(nx, ny) == 0 {
			return Point{nx, ny}, true
		}
	}
	return Point{}, false
}

func (g *NavGrid) ResolveTarget(x, y int) (Point, bool) {
	if g == nil || g.raster == nil || !g.raster.InBounds(x, y) {
		return Point{}, false
	}

	tx, ty := x, y
	if g.pixelCellOpen(tx, ty) {
		if !g.raster.Walkable(tx, ty) {
			return Point{}, false
		}
		return Point{tx, ty}, true
	}

	found := false
	for yy := y; yy < g.raster.Height-4; yy++ {
		if g.pixelCellOpen(x, yy) {
			ty = yy
			found = true
			break
		}
	}
	if !found {
		for yy := y; yy > 4; yy-- {
			if g.pixelCellOpen(x, yy) {
				ty = yy
				found = true
				break
			}
		}
	}
	if !found {
		ty = y
	}

	foundX := false
	for xx := x; xx < g.raster.Width-4; xx++ {
		if g.pixelCellOpen(xx, ty) {
			tx = xx
			foundX = true
			break
		}
	}
	if !foundX {
		for xx := x; xx > 4; xx-- {
			if g.pixelCellOpen(xx, ty) {
				tx = xx
				foundX = true
				break
			}
		}
	}
	if !foundX {
		return Point{}, false
	}
	if !g.raster.InBounds(tx, ty) || !g.raster.Walkable(tx, ty) {
		return Point{}, false
	}
	return Point{tx, ty}, true
}

func (g *NavGrid) pixelCellOpen(x, y int) bool {
	cx, cy := g.CellAt(x, y)
	return g.cellFlags(cx, cy) == 0
}

func NearestWalkablePixel(raster *zbf.Raster, x, y, maxRadius int) (Point, bool) {
	if raster.InBounds(x, y) && raster.Walkable(x, y) {
		return Point{x, y}, true
	}
	for radius := 1; radius <= maxRadius; radius++ {
		var best Point
		bestDist := math.MaxFloat64
		found := false
		check := func(px, py int) {
			if raster.InBounds(px, py) && raster.Walkable(px, py) {
				d := math.Hypot(float64(px-x), float64(py-y))
				if d < bestDist {
					bestDist = d
					best = Point{px, py}
					found = true
				}
			}
		}
		for dx := -radius; dx <= radius; dx++ {
			check(x+dx, y-radius)
			check(x+dx, y+radius)
		}
		for dy := -radius + 1; dy <= radius-1; dy++ {
			check(x-radius, y+dy)
			check(x+radius, y+dy)
		}
		if found {
			return best, true
		}
	}
	return Point{}, false
}

type pathNode struct {
	x, y   int
	parent uint16
	g, h   int
}

var neighborOffsets = [...]Point{
	{0, -1},
	{1, -1},
	{1, 0},
	{1, 1},
	{0, 1},
	{-1, 1},
	{-1, 0},
	{-1, -1},
}

var neighborCosts = [...]int{1000, 1414, 1000, 1414, 1000, 1414, 1000, 1414}

func (g *NavGrid) FindPath(start, goal Point) ([]Point, bool) {
	g.clearVisited()

	startCX, startCY := g.CellAt(start.X, start.Y)
	goalCX, goalCY := g.CellAt(goal.X, goal.Y)

	startCell, ok := g.adjustBlockedCell(startCX, startCY)
	if !ok {
		return nil, false
	}
	goalCell, ok := g.adjustBlockedCell(goalCX, goalCY)
	if !ok {
		return nil, false
	}

	if startCell == goalCell {
		return []Point{start, goal}, true
	}

	nodes, goalID, ok := g.searchOriginal(goalCell, startCell)
	if !ok {
		return nil, false
	}

	if startCell.X != startCX || startCell.Y != startCY {
		sx, sy := g.CellCenter(startCell.X, startCell.Y)
		snappedStart := Point{sx, sy}
		path := g.buildOriginalWaypoints(snappedStart, goal, nodes, goalID)
		return append([]Point{start}, path...), true
	}

	return g.buildOriginalWaypoints(start, goal, nodes, goalID), true
}

func (g *NavGrid) searchOriginal(searchStart, searchGoal Point) ([]pathNode, uint16, bool) {
	nodes := make([]pathNode, 2, 256)
	nodes[1] = pathNode{
		x: searchStart.X,
		y: searchStart.Y,
		g: 0,
		h: originalHeuristic(searchStart, searchGoal),
	}
	open := []uint16{1}
	closed := make([]uint16, 0, 256)
	g.flags[g.index(searchStart.X, searchStart.Y)] |= navVisited

	for len(open) > 0 {
		currentID := open[0]
		open = open[1:]
		current := nodes[currentID]
		if current.x == searchGoal.X && current.y == searchGoal.Y {
			g.clearVisited()
			return nodes, currentID, true
		}

		for i, off := range neighborOffsets {
			nx := current.x + off.X
			ny := current.y + off.Y
			if g.cellFlags(nx, ny)&0xc0 != 0 {
				continue
			}

			newG := current.g + neighborCosts[i]
			_, openID := findNode(nodes, open, nx, ny)
			_, closedID := findNode(nodes, closed, nx, ny)

			if openID != 0 && newG >= nodes[openID].g {
				continue
			}
			if openID == 0 && closedID != 0 {
				continue
			}

			h := originalHeuristic(Point{nx, ny}, searchGoal)
			if g.cellFlags(nx, ny)&navVisited == 0 {
				nodes = append(nodes, pathNode{x: nx, y: ny, parent: currentID, g: newG, h: h})
				openID = uint16(len(nodes) - 1)
				g.flags[g.index(nx, ny)] |= navVisited
			} else {
				if openID == 0 {
					continue
				}
				nodes[openID].parent = currentID
				nodes[openID].g = newG
				nodes[openID].h = h
			}

			if openID != 0 {
				openPos, existing := findNode(nodes, open, nx, ny)
				if existing != 0 {
					open = append(open[:openPos], open[openPos+1:]...)
				}
			}
			open = insertOpen(nodes, open, openID)
		}

		closed = append([]uint16{currentID}, closed...)
	}

	g.clearVisited()
	return nil, 0, false
}

func findNode(nodes []pathNode, ids []uint16, x, y int) (int, uint16) {
	for i, id := range ids {
		n := nodes[id]
		if n.x == x && n.y == y {
			return i, id
		}
	}
	return -1, 0
}

func insertOpen(nodes []pathNode, open []uint16, id uint16) []uint16 {
	f := nodes[id].g + nodes[id].h
	at := 0
	for at < len(open) {
		other := nodes[open[at]]
		if f <= other.g+other.h {
			break
		}
		at++
	}
	open = append(open, 0)
	copy(open[at+1:], open[at:])
	open[at] = id
	return open
}

func (g *NavGrid) buildOriginalWaypoints(start, goal Point, nodes []pathNode, actorNodeID uint16) []Point {
	out := []Point{start}
	nextID := nodes[actorNodeID].parent
	anchorX := start.X
	anchorY := start.Y

	for {
		if nextID == 0 {
			return append(out, goal)
		}

		anchorCellX := anchorX / g.cellSize
		anchorCellY := anchorY / g.cellSize
		prevX := anchorX
		prevY := anchorY

		for nextID != 0 {
			n := nodes[nextID]
			candidateX := n.x*g.cellSize + 2
			candidateY := n.y*g.cellSize + 2
			if !g.hasCellLineOfSight(Point{anchorCellX, anchorCellY}, Point{n.x, n.y}) {
				out = append(out, Point{prevX, prevY})
				anchorX = prevX
				anchorY = prevY
				break
			}
			prevX = candidateX
			prevY = candidateY
			nextID = n.parent
		}

		if nextID == 0 {
			return append(out, goal)
		}
	}
}

func originalHeuristic(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return int(math.Sqrt(float64(dx*dx+dy*dy)) * 1000.0)
}

func (g *NavGrid) hasCellLineOfSight(a, b Point) bool {
	dx := b.X - a.X
	dy := b.Y - a.Y
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		return true
	}

	x := a.X << 10
	y := a.Y << 10
	xStep := (dx << 10) / steps
	yStep := (dy << 10) / steps
	for i := 1; i <= steps; i++ {
		if g.cellFlags(x>>10, y>>10) != 0 {
			return false
		}
		x += xStep
		y += yStep
	}
	return true
}

func hasLineOfSight(raster *zbf.Raster, a, b Point) bool {
	x0, y0 := a.X, a.Y
	x1, y1 := b.X, b.Y
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if !raster.InBounds(x0, y0) || !raster.Walkable(x0, y0) {
			return false
		}
		if x0 == x1 && y0 == y1 {
			return true
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
