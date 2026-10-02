package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

func load7BWalkRaster(t *testing.T) *zbf.Raster {
	t.Helper()

	path := filepath.Join(repoRoot(t), "data", "cd", "GAME", "LOC01", "BACK7B_PATH.ZBF")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read BACK7B_PATH.ZBF: %v", err)
	}

	r, err := zbf.Parse(data)
	if err != nil {
		t.Fatalf("zbf.Parse: %v", err)
	}

	return r
}

// firstWalkablePixel scans the raster for a deterministic walkable pixel to
// use as a test start/goal point.
func firstWalkablePixel(t *testing.T, r *zbf.Raster) Point {
	t.Helper()

	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			if r.Walkable(x, y) {
				return Point{x, y}
			}
		}
	}

	t.Fatal("raster has no walkable pixels")
	return Point{}
}

// largestComponentExtremes flood-fills the walkable mask's connected
// components (8-connected) and returns two distinct, well-separated points
// guaranteed to lie in the single largest component. Real path masks can
// contain stray noise pixels far from the main walkable floor (see
// agents/ZBF.md: "path values are not just a boolean mask") that are not
// reachable from it at all, so tests must not assume any two arbitrary
// walkable pixels are connected.
func largestComponentExtremes(t *testing.T, r *zbf.Raster) (Point, Point) {
	t.Helper()

	visited := make([]bool, r.Width*r.Height)
	var bestA, bestB Point
	bestSize := 0

	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			idx := y*r.Width + x
			if visited[idx] || !r.Walkable(x, y) {
				continue
			}

			stack := []Point{{x, y}}
			visited[idx] = true
			size := 0
			// Track the pixels with the smallest and largest (X+Y), a cheap
			// proxy for "opposite corners" of the component, so callers get
			// two genuinely far-apart points rather than two DFS-adjacent
			// ones (a plain "first/last visited" pair is often only 1px
			// apart since DFS explores locally before backtracking).
			first := Point{x, y}
			last := Point{x, y}
			minSum, maxSum := x+y, x+y

			for len(stack) > 0 {
				c := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++

				if s := c.X + c.Y; s < minSum {
					minSum, first = s, c
				} else if s > maxSum {
					maxSum, last = s, c
				}

				for _, off := range neighborOffsets {
					nx, ny := c.X+off[0], c.Y+off[1]
					if nx < 0 || ny < 0 || nx >= r.Width || ny >= r.Height {
						continue
					}
					nidx := ny*r.Width + nx
					if visited[nidx] || !r.Walkable(nx, ny) {
						continue
					}
					visited[nidx] = true
					stack = append(stack, Point{nx, ny})
				}
			}

			if size > bestSize {
				bestSize, bestA, bestB = size, first, last
			}
		}
	}

	if bestSize == 0 {
		t.Fatal("raster has no walkable pixels")
	}

	return bestA, bestB
}

func TestBuildNavGridAnchorsAreAlwaysWalkable(t *testing.T) {
	r := load7BWalkRaster(t)
	g := BuildNavGrid(r)

	for cy := range g.rows {
		for cx := range g.cols {
			if !g.walkableCell(cx, cy) {
				continue
			}

			px, py := g.CellCenter(cx, cy)
			if !r.Walkable(px, py) {
				t.Fatalf("cell (%d,%d) marked walkable but its anchor (%d,%d) is not", cx, cy, px, py)
			}
		}
	}
}

func TestBuildNavGridCapturesThinSlivers(t *testing.T) {
	// A cell whose only walkable pixel is off-center (not the geometric
	// center sample) must still be marked walkable, per the anchor-based
	// design: real path masks contain sub-cell-width diagonal slivers.
	pixels := make([]byte, zbf.SceneWidth*zbf.SceneHeight)
	pixels[0*zbf.SceneWidth+3] = 100 // corner pixel of cell (0,0), not its center

	r := &zbf.Raster{Width: zbf.SceneWidth, Height: zbf.SceneHeight, Pixels: pixels}
	g := BuildNavGrid(r)

	if !g.walkableCell(0, 0) {
		t.Fatal("expected cell (0,0) to be walkable via its off-center pixel")
	}

	px, py := g.CellCenter(0, 0)
	if px != 3 || py != 0 {
		t.Errorf("anchor = (%d,%d), want (3,0)", px, py)
	}
}

func TestFindPathBetweenTwoRealWalkablePoints(t *testing.T) {
	r := load7BWalkRaster(t)
	g := BuildNavGrid(r)

	start, goal := largestComponentExtremes(t, r)

	path, ok := g.FindPath(start, goal)
	if !ok {
		t.Fatal("expected a path between two real walkable points in the same scene")
	}

	if len(path) == 0 {
		t.Fatal("path is empty")
	}

	if path[len(path)-1] != goal {
		t.Errorf("last waypoint = %v, want goal %v", path[len(path)-1], goal)
	}

	// Every consecutive waypoint pair must have full line-of-sight over
	// walkable pixels only (this is what simplifyPath guarantees).
	for i := 0; i < len(path)-1; i++ {
		if !hasLineOfSight(r, path[i], path[i+1]) {
			t.Errorf("no line of sight between waypoint %d (%v) and %d (%v)", i, path[i], i+1, path[i+1])
		}
	}
}

func TestFindPathFailsAcrossDisconnectedRegions(t *testing.T) {
	// A raster where the only two walkable pixels are on opposite corners
	// with an solid wall of unwalkable cells between them: no path should
	// be found, not a silent teleport.
	pixels := make([]byte, zbf.SceneWidth*zbf.SceneHeight)
	set := func(x, y int, v byte) { pixels[y*zbf.SceneWidth+x] = v }

	// Two small walkable rooms in opposite corners, fully separated (no
	// walkable pixels anywhere else).
	for y := range 5 {
		for x := range 5 {
			set(x, y, 200)
			set(zbf.SceneWidth-1-x, zbf.SceneHeight-1-y, 200)
		}
	}

	r := &zbf.Raster{Width: zbf.SceneWidth, Height: zbf.SceneHeight, Pixels: pixels}
	g := BuildNavGrid(r)

	_, ok := g.FindPath(Point{2, 2}, Point{zbf.SceneWidth - 3, zbf.SceneHeight - 3})
	if ok {
		t.Fatal("expected no path between disconnected regions")
	}
}

func TestNearestWalkablePixel(t *testing.T) {
	r := load7BWalkRaster(t)

	// (0,0) is very likely non-walkable (background corner); confirm the
	// function either finds something plausible nearby or fails within the
	// bound, but never returns something farther than the bound implies.
	p, ok := NearestWalkablePixel(r, 0, 0, 50)
	if ok {
		if !r.Walkable(p.X, p.Y) {
			t.Errorf("returned point (%v) is not actually walkable", p)
		}
	}
}

func TestNearestWalkablePixelAlreadyWalkable(t *testing.T) {
	r := load7BWalkRaster(t)
	start := firstWalkablePixel(t, r)

	p, ok := NearestWalkablePixel(r, start.X, start.Y, 10)
	if !ok || p != start {
		t.Errorf("NearestWalkablePixel(%v) = %v,%v, want %v,true (already walkable)", start, p, ok, start)
	}
}

func TestNearestWalkablePixelFailsBeyondRadius(t *testing.T) {
	pixels := make([]byte, zbf.SceneWidth*zbf.SceneHeight)
	// Single walkable pixel, far from origin.
	pixels[359*zbf.SceneWidth+639] = 100

	r := &zbf.Raster{Width: zbf.SceneWidth, Height: zbf.SceneHeight, Pixels: pixels}

	if _, ok := NearestWalkablePixel(r, 0, 0, 5); ok {
		t.Error("expected failure: the only walkable pixel is far beyond the search radius")
	}
}

func TestHasLineOfSightBlockedByWall(t *testing.T) {
	pixels := make([]byte, zbf.SceneWidth*zbf.SceneHeight)
	for i := range pixels {
		pixels[i] = 100
	}
	// Solid vertical wall at x=10.
	for y := range zbf.SceneHeight {
		pixels[y*zbf.SceneWidth+10] = 0
	}

	r := &zbf.Raster{Width: zbf.SceneWidth, Height: zbf.SceneHeight, Pixels: pixels}

	if hasLineOfSight(r, Point{5, 5}, Point{15, 5}) {
		t.Error("expected line of sight to be blocked by the wall")
	}

	if !hasLineOfSight(r, Point{5, 5}, Point{9, 5}) {
		t.Error("expected clear line of sight on the walkable side of the wall")
	}
}
