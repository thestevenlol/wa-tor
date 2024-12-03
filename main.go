package main

import (
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/exp/rand"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth     = 1000
	screenHeight    = 1000
	gridSize        = screenWidth / cellSize
	cellSize        = 3
	fishBreedTime   = 3
	sharkBreedTime  = 8
	sharkStarveTime = 3
	fishPercentage  = 50
	sharkPercentage = 20
	logFilePrefix   = "tpsMeasurementNThreads"
	nThreads        = 7
)

type CellType int

const (
	Empty CellType = iota
	Fish
	Shark
)

var rowHeights map[int][2]int
var wg sync.WaitGroup

// Cell struct
// Parameters: Type, BreedTime, StarveTime
// Returns: None
// Description: Cell struct to represent a cell in the grid
type Cell struct {
	Type       CellType
	BreedTime  int
	StarveTime int
}

type Game struct {
	grid [gridSize][gridSize]Cell
}

func ProcessRow(yMin, yMax int, game *Game, wg *sync.WaitGroup) {
	defer wg.Done()

	// Add bounds checking
	if yMin < 0 {
		yMin = 0
	}
	if yMax > gridSize {
		yMax = gridSize
	}

	newGrid := [gridSize][gridSize]Cell{}
	moved := make(map[[2]int]bool)

	// First process sharks in this row range
	for y := yMin; y < yMax; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			if game.grid[y][x].Type == Shark {
				game.updateShark(x, y, &newGrid, moved)
			}
		}
	}

	// Then process fish in this row range
	for y := yMin; y < yMax; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			if game.grid[y][x].Type == Fish {
				game.updateFish(x, y, &newGrid, moved)
			}
		}
	}

	// Update only this thread's section of the main grid
	for y := yMin; y < yMax; y++ {
		for x := 0; x < gridSize; x++ {
			if newGrid[y][x].Type != Empty {
				game.grid[y][x] = newGrid[y][x]
			}
		}
	}
}

func AssignThreads(heightMap map[int][2]int, game *Game, wg *sync.WaitGroup) {
	for _, minmax := range heightMap {
		wg.Add(1)
		go ProcessRow(minmax[0], minmax[1], game, wg)
	}
	wg.Wait()
}

func GetRowHeights() map[int][2]int {
	heights := make(map[int][2]int)
	rowsPerThread := gridSize / nThreads // Changed from screenHeight to gridSize

	for i := 0; i < nThreads; i++ {
		yMin := i * rowsPerThread
		yMax := (i + 1) * rowsPerThread
		if i == nThreads-1 {
			yMax = gridSize // Changed from screenHeight to gridSize
		}
		heights[i] = [2]int{yMin, yMax}
	}
	return heights
}

// Initialise function
// Parameters: None
// Returns: None
// Description: Initialises the game grid by randomly placing fish and sharks
func (g *Game) Initialise() {
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			random := rand.Intn(100)

			if random < fishPercentage {
				g.grid[y][x] = Cell{Type: Fish, BreedTime: fishBreedTime}
			} else if random < (fishPercentage + sharkPercentage) {
				g.grid[y][x] = Cell{Type: Shark, BreedTime: sharkBreedTime, StarveTime: sharkStarveTime}
			} else {
				g.grid[y][x] = Cell{Type: Empty}
			}
		}
	}
}

// GetAdjacent function
// Parameters: x, y
// Returns: [][2]int
// Description: Returns the adjacent cells to the given cell
func (g *Game) GetAdjacent(x, y int) [][2]int {
	adjacent := make([][2]int, 0, 4)
	directions := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	/*

		current cell: (5, 10)
		adjacent cells: (4, 10), (6, 10), (5, 9), (5, 11)

	*/

	for _, d := range directions {
		newX := (x + d[0] + gridSize) % gridSize
		newY := (y + d[1] + gridSize) % gridSize
		adjacent = append(adjacent, [2]int{newX, newY})
	}
	return adjacent
}

// Shuffle function
// Parameters: slice
// Returns: None
// Description: Shuffles the slice in place using the Fisher-Yates algorithm
func Shuffle(slice [][2]int) {
	rand.Seed(uint64(time.Now().UnixNano()))

	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)                   // Generate a random index
		slice[i], slice[j] = slice[j], slice[i] // Swap elements
	}
}

// Extract shark update logic
func (g *Game) updateShark(x, y int, newGrid *[gridSize][gridSize]Cell, moved map[[2]int]bool) {
	cell := g.grid[y][x]
	adjacent := g.GetAdjacent(x, y)
	Shuffle(adjacent)

	// Look for fish to eat
	fishFound := false
	for _, pos := range adjacent {
		if g.grid[pos[1]][pos[0]].Type == Fish && !moved[pos] {
			// Shark eats fish and moves
			(*newGrid)[pos[1]][pos[0]] = Cell{
				Type:       Shark,
				BreedTime:  cell.BreedTime - 1,
				StarveTime: sharkStarveTime,
			}
			moved[pos] = true
			fishFound = true
			break
		}
	}

	if !fishFound {
		// Move to empty space if no fish found
		emptySpaces := make([][2]int, 0)
		for _, pos := range adjacent {
			if g.grid[pos[1]][pos[0]].Type == Empty && !moved[pos] {
				emptySpaces = append(emptySpaces, pos)
			}
		}

		if len(emptySpaces) > 0 {
			newPos := emptySpaces[rand.Intn(len(emptySpaces))]
			cell.StarveTime--
			cell.BreedTime--

			if cell.StarveTime <= 0 {
				(*newGrid)[newPos[1]][newPos[0]] = Cell{Type: Empty}
			} else {
				if cell.BreedTime <= 0 {
					// Reproduce
					(*newGrid)[y][x] = Cell{
						Type:       Shark,
						BreedTime:  sharkBreedTime,
						StarveTime: sharkStarveTime,
					}
					cell.BreedTime = sharkBreedTime
				}
				(*newGrid)[newPos[1]][newPos[0]] = cell
			}
			moved[newPos] = true
		} else {
			(*newGrid)[y][x] = cell
		}
	}
}

// Extract fish update logic
func (g *Game) updateFish(x, y int, newGrid *[gridSize][gridSize]Cell, moved map[[2]int]bool) {
	cell := g.grid[y][x]
	adjacent := g.GetAdjacent(x, y)
	emptySpaces := make([][2]int, 0)

	for _, pos := range adjacent {
		if g.grid[pos[1]][pos[0]].Type == Empty && !moved[pos] {
			emptySpaces = append(emptySpaces, pos)
		}
	}

	if len(emptySpaces) > 0 {
		newPos := emptySpaces[rand.Intn(len(emptySpaces))]
		cell.BreedTime--

		if cell.BreedTime <= 0 {
			// Reproduce
			(*newGrid)[y][x] = Cell{
				Type:      Fish,
				BreedTime: fishBreedTime,
			}
			cell.BreedTime = fishBreedTime
		}
		(*newGrid)[newPos[1]][newPos[0]] = cell
		moved[newPos] = true
	} else {
		(*newGrid)[y][x] = cell
	}
}

// Modified Update function
func (g *Game) Update() error {
	// Create waitgroup for synchronization
	var wg sync.WaitGroup

	// Get thread assignments
	heights := GetRowHeights()

	// Process all rows using threads
	AssignThreads(heights, g, &wg)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			cell := g.grid[y][x]
			var colour color.Color
			switch cell.Type {
			case Fish:
				colour = color.RGBA{R: 0, G: 180, B: 0, A: 255}
			case Shark:
				colour = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			default:
				continue
			}
			vector.DrawFilledRect(screen, float32(x*cellSize), float32(y*cellSize), cellSize, cellSize, colour, false)
		}
	}

	// Get the current FPS
	tps := ebiten.ActualTPS()

	// Convert the FPS value to a string
	tpsString := fmt.Sprintf("FPS: %.2f", tps)

	// Draw the FPS value on the screen at the top-left corner
	text.Draw(screen, tpsString, basicfont.Face7x13, 10, 20, color.White)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {

	rowHeights = GetRowHeights()
	for _, rh := range rowHeights {
		fmt.Println(rh)
	}

	rand.Seed(uint64(time.Now().UnixNano()))
	game := &Game{}
	game.Initialise()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Wator Simulation in Go! (Ebiten)")
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
