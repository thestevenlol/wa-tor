package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/exp/rand"
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
	nThreads        = 16
	logFilePrefix   = "tpsMeasurementNThreads"
	nRows           = screenHeight / nThreads
)

// Cell types
const (
	Empty int32 = 0
	Fish  int32 = 1
	Shark int32 = 2
)

// Global variables
var threads []ThreadBounds
var current time.Time
var occupied [gridSize][gridSize]int32

// ThreadBounds struct
// Parameters: MinY, MaxY
// Returns: None
// Description: ThreadBounds struct to represent the bounds of a thread
type ThreadBounds struct {
	MinY int
	MaxY int
}

// ThreadGrid struct
// Parameters: minY, maxY, isBoundary
// Returns: None
// Description: ThreadGrid struct to represent a grid for a thread
type ThreadGrid struct {
	minY, maxY int
	isBoundary [][2]bool // [y][x]bool to mark boundary cells
}

// Cell struct
// Parameters: Type, BreedTime, StarveTime
// Returns: None
// Description: Cell struct to represent a cell in the grid
type Cell struct {
	Type       int32 // Regular int32, we'll use atomic operations to access it
	BreedTime  int
	StarveTime int
}

// Game struct
// Parameters: grid
// Returns: None
// Description: Game struct to represent the game
type Game struct {
	grid [gridSize][gridSize]Cell
}

// NewThreadGrid function
// Parameters: bounds
// Returns: *ThreadGrid
// Description: Returns a new ThreadGrid with the given bounds
func NewThreadGrid(bounds ThreadBounds) *ThreadGrid {
	tg := &ThreadGrid{
		minY:       bounds.MinY,
		maxY:       bounds.MaxY,
		isBoundary: make([][2]bool, gridSize),
	}
	// Initialize boundary markers
	for i := 0; i < gridSize; i++ {
		tg.isBoundary[i] = [2]bool{bounds.MinY == 0, bounds.MaxY == gridSize}
	}
	return tg
}

// isThreadBoundary function
// Parameters: y
// Returns: bool
// Description: Returns true if the given y is a boundary of the thread
func (tg *ThreadGrid) isThreadBoundary(y int) bool {
	return y == tg.minY || y == tg.maxY-1
}

// Initialise function
// Parameters: None
// Returns: None
// Description: Initialises the game grid by randomly placing fish and sharks
func (g *Game) Initialise() {
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			random := rand.Intn(100)
			var cell Cell
			if random < fishPercentage {
				cell.Type = Fish
				cell.BreedTime = fishBreedTime
			} else if random < (fishPercentage + sharkPercentage) {
				cell.Type = Shark
				cell.BreedTime = sharkBreedTime
				cell.StarveTime = sharkStarveTime
			} else {
				cell.Type = Empty
			}

			g.grid[y][x] = cell
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
// Description: Shuffles the slice in place
func Shuffle(slice [][2]int) {
	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)                   // Generate a random index
		slice[i], slice[j] = slice[j], slice[i] // Swap elements
	}
}

// updateShark function
// Parameters: newGrid, tg, occupied, g
// Returns: error
// Description: Updates the shark cells in the grid
func updateShark(newGrid *[gridSize][gridSize]Cell, tg *ThreadGrid, occupied *[gridSize][gridSize]int32, g *Game) error {
	for y := tg.minY; y < tg.maxY; y++ {
		for x := 0; x < gridSize; x++ {
			if tg.isThreadBoundary(y) {
				if atomic.LoadInt32(&occupied[y][x]) == 1 {
					continue
				}
			} else if occupied[y][x] == 1 {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Shark {
				adjacent := g.GetAdjacent(x, y)
				Shuffle(adjacent)

				// Look for fish to eat
				fishFound := false
				for _, pos := range adjacent {
					if atomic.LoadInt32(&g.grid[pos[1]][pos[0]].Type) == Fish {
						if atomic.CompareAndSwapInt32(&occupied[pos[1]][pos[0]], 0, 1) {
							// Shark eats fish and moves
							newGrid[pos[1]][pos[0]] = Cell{
								Type:       Shark,
								BreedTime:  cell.BreedTime - 1,
								StarveTime: sharkStarveTime,
							}
						} else {
							continue
						}
					}
				}

				if !fishFound {
					// Move to empty space if no fish found
					emptySpaces := make([][2]int, 0)
					for _, pos := range adjacent {
						if atomic.LoadInt32(&g.grid[pos[1]][pos[0]].Type) == Empty {
							if atomic.CompareAndSwapInt32(&occupied[y][x], 0, 1) {
								emptySpaces = append(emptySpaces, pos)
							}
						}
					}

					if len(emptySpaces) > 0 {
						newPos := emptySpaces[rand.Intn(len(emptySpaces))]
						cell.StarveTime--
						cell.BreedTime--

						if cell.StarveTime <= 0 {
							newGrid[newPos[1]][newPos[0]] = Cell{Type: Empty}
						} else {
							if cell.BreedTime <= 0 {
								// Reproduce
								newGrid[y][x] = Cell{
									Type:       Shark,
									BreedTime:  sharkBreedTime,
									StarveTime: sharkStarveTime,
								}
								cell.BreedTime = sharkBreedTime
							}
							newGrid[newPos[1]][newPos[0]] = cell
						}
						// When moving
						newGrid[y][x] = Cell{Type: Empty}
						if !atomic.CompareAndSwapInt32(&occupied[newPos[1]][newPos[0]], 0, 1) {
							continue
						}
					} else {
						newGrid[y][x] = cell
					}
				}
			}
		}
	}
	return nil
}

// updateFish function
// Parameters: newGrid, tg, occupied, g
// Returns: error
// Description: Updates the fish cells in the grid
func updateFish(newGrid *[gridSize][gridSize]Cell, tg *ThreadGrid, occupied *[gridSize][gridSize]int32, g *Game) error {
	for y := tg.minY; y < tg.maxY; y++ {
		for x := 0; x < gridSize; x++ {
			if tg.isThreadBoundary(y) {
				if atomic.LoadInt32(&occupied[y][x]) == 1 {
					continue
				}
			} else if occupied[y][x] == 1 {
				continue
			}

			cell := g.grid[y][x]
			cellType := atomic.LoadInt32(&cell.Type)
			if cellType == Fish {
				adjacent := g.GetAdjacent(x, y)
				Shuffle(adjacent)
				emptySpaces := make([][2]int, 0)

				for _, pos := range adjacent {
					if atomic.LoadInt32(&g.grid[pos[1]][pos[0]].Type) == Empty {
						if atomic.CompareAndSwapInt32(&occupied[pos[1]][pos[0]], 0, 1) {
							emptySpaces = append(emptySpaces, pos)
						}
					}
				}

				if len(emptySpaces) > 0 {
					newPos := emptySpaces[rand.Intn(len(emptySpaces))]
					cell.BreedTime--

					if cell.BreedTime <= 0 {
						// Reproduce
						atomic.StoreInt32(&newGrid[y][x].Type, Fish)
						newGrid[y][x].BreedTime = fishBreedTime
						cell.BreedTime = fishBreedTime
					} else {
						atomic.StoreInt32(&newGrid[y][x].Type, Empty)
					}

					atomic.StoreInt32(&newGrid[newPos[1]][newPos[0]].Type, Fish)
					newGrid[newPos[1]][newPos[0]].BreedTime = cell.BreedTime

					if !atomic.CompareAndSwapInt32(&occupied[newPos[1]][newPos[0]], 0, 1) {
						continue
					}
				} else {
					newGrid[y][x] = cell
				}
			}
		}
	}
	return nil
}

// Update function
// Parameters: None
// Returns: error
// Description: Updates the game state
func (g *Game) Update() error {
	// Add at start of Update():
	for i := range occupied {
		for j := range occupied[i] {
			atomic.StoreInt32(&occupied[i][j], 0)
		}
	}

	newGrid := &[gridSize][gridSize]Cell{}
	var wg sync.WaitGroup

	if time.Since(current).Milliseconds() > 500 {
		WriteTPS()
		current = time.Now()
	}

	threadGrids := make([]*ThreadGrid, len(threads))
	for i, thread := range threads {
		threadGrids[i] = NewThreadGrid(thread)
	}

	for _, tg := range threadGrids {
		wg.Add(2)
		go func(tg *ThreadGrid) {
			defer wg.Done()
			updateShark(newGrid, tg, &occupied, g)
		}(tg)
		go func(tg *ThreadGrid) {
			defer wg.Done()
			updateFish(newGrid, tg, &occupied, g)
		}(tg)
	}

	wg.Wait()
	g.grid = *newGrid
	return nil
}

// Draw function
// Parameters: screen
// Returns: None
// Description: Draws the game state to the screen
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			cellType := atomic.LoadInt32(&g.grid[y][x].Type)
			var colour color.Color
			switch cellType {
			case Fish:
				colour = color.RGBA{R: 0, G: 180, B: 0, A: 255}
			case Shark:
				colour = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			default:
				continue
			}

			ebitenutil.DrawRect(
				screen,
				float64(x*cellSize),
				float64(y*cellSize),
				float64(cellSize),
				float64(cellSize),
				colour,
			)
		}
	}
}

// Layout function
// Parameters: outsideWidth, outsideHeight
// Returns: int, int
// Description: Returns the screen width and height
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// WriteTPS function
// Parameters: None
// Returns: None
// Description: Writes the current TPS to a CSV file
func WriteTPS() {
	file, err := os.OpenFile(fmt.Sprintf("%s_%d.csv", logFilePrefix, nThreads), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers if file is empty
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write([]string{"tps"}); err != nil {
			log.Fatal(err)
		}
	}

	// Write current TPS
	tps := ebiten.ActualTPS()
	if err := writer.Write([]string{fmt.Sprintf("%.2f", tps)}); err != nil {
		log.Fatal(err)
	}
}

// GetThreadRowHeights function
// Parameters: None
// Returns: []int
// Description: Returns the heights of each thread
func GetThreadRowHeights() []int {
	// Calculate base height per thread
	baseHeight := gridSize / nThreads
	remainder := gridSize % nThreads

	// Distribute heights
	heights := make([]int, nThreads)
	for i := range heights {
		heights[i] = baseHeight
		// Distribute remainder one extra row at a time
		if remainder > 0 {
			heights[i]++
			remainder--
		}
	}

	return heights
}

// GetThreadYBounds function
// Parameters: None
// Returns: []ThreadBounds
// Description: Returns the bounds of each thread
func GetThreadYBounds() []ThreadBounds {
	heights := GetThreadRowHeights()
	bounds := make([]ThreadBounds, len(heights))

	currentY := 0
	for i, height := range heights {
		bounds[i] = ThreadBounds{
			MinY: currentY,
			MaxY: currentY + height,
		}
		currentY += height
	}

	return bounds
}

// main function
// Parameters: None
// Returns: None
// Description: Main function to run the simulation
func main() {
	threads = GetThreadYBounds()
	current = time.Now()
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
