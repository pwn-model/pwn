package maps

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/backends/opengl"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// Trees visualizes the trees grid as a color map.
type Trees struct {
	treeFilter    *ecs.Filter1[comp.Position]
	damagedFilter *ecs.Filter1[comp.Position]

	grid   ecs.Resource[res.EntityGrid]
	image  *image.RGBA
	canvas *opengl.Canvas
}

// Initialize the drawer.
func (s *Trees) Initialize(world *ecs.World, _ *opengl.Window) {
	s.treeFilter = s.treeFilter.New(world).Without(ecs.C[comp.Damaged]())
	s.damagedFilter = s.damagedFilter.New(world).With(ecs.C[comp.Damaged]())

	s.grid = s.grid.New(world)
	grid := s.grid.Get()
	s.image = image.NewRGBA(image.Rect(0, 0, grid.Width(), grid.Width()))
	s.canvas = opengl.NewCanvas(pixel.R(0, 0, float64(grid.Width()), float64(grid.Width())))
}

// Update the drawer.
func (s *Trees) Update(_ *ecs.World) {}

// UpdateInputs of the drawer.
func (s *Trees) UpdateInputs(_ *ecs.World, _ *opengl.Window) {}

// Draw the drawer's content.
func (s *Trees) Draw(world *ecs.World, win *opengl.Window) {
	grid := s.grid.Get()
	draw.Draw(s.image, s.image.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	white, red := color.White, color.RGBA{255, 0, 0, 255}

	q := s.treeFilter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.image.Set(pos.X, pos.Y, white)
		}
	}

	q = s.damagedFilter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.image.Set(pos.X, pos.Y, red)
		}
	}

	s.canvas.SetPixels(s.image.Pix)

	scale := math.Min(
		float64(win.Bounds().W())/float64(grid.Width()),
		float64(win.Bounds().H())/float64(grid.Height()),
	)

	offset := pixel.V(
		float64(win.Bounds().W()/2),
		float64(win.Bounds().H()/2),
	)

	s.canvas.Draw(win, pixel.IM.Scaled(pixel.ZV, scale).Moved(offset))
}
