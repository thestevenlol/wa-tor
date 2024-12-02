package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
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
	logFilePrefix   = "tpsMeasurementNThreads"
	nThreads        = 4
	nRows           = screenHeight / nThreads
)

type CellType int

const (
	Empty CellType = iota
	Fish
	Shark
)

type ThreadBounds struct {
	MinY int
	MaxY int
}

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

var threads []ThreadBounds
var current time.Time

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
// Description: Shuffles the slice in place
func Shuffle(slice [][2]int) {
	rand.Seed(uint64(time.Now().UnixNano()))

	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)                   // Generate a random index
		slice[i], slice[j] = slice[j], slice[i] // Swap elements
	}
}

func updateShark(newGrid *[gridSize][gridSize]Cell, yMin, yMax int, moved *SafeMap, g *Game) error {
	for y := yMin; y < yMax; y++ {
		for x := 0; x < gridSize; x++ {
			if val, _ := moved.Get([2]int{x, y}); val {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Shark {
				adjacent := g.GetAdjacent(x, y)
				Shuffle(adjacent)

				// Look for fish to eat
				fishFound := false
				for _, pos := range adjacent {
					if g.grid[pos[1]][pos[0]].Type == Fish {
						if val, _ := moved.Get(pos); !val {
							// Shark eats fish and moves
							newGrid[pos[1]][pos[0]] = Cell{
								Type:       Shark,
								BreedTime:  cell.BreedTime - 1,
								StarveTime: sharkStarveTime,
							}
							moved.Set(pos, true)
							fishFound = true
							break
						}
					}
				}

				if !fishFound {
					// Move to empty space if no fish found
					emptySpaces := make([][2]int, 0)
					for _, pos := range adjacent {
						if g.grid[pos[1]][pos[0]].Type == Empty {
							if val, _ := moved.Get(pos); !val {
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
						moved.Set(newPos, true)
					} else {
						newGrid[y][x] = cell
					}
				}
			}
		}
	}
	return nil
}

func updateFish(newGrid *[gridSize][gridSize]Cell, yMin, yMax int, moved *SafeMap, g *Game) {
	for y := yMin; y < yMax; y++ {
		for x := 0; x < gridSize; x++ {
			if val, _ := moved.Get([2]int{x, y}); val {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Fish {
				adjacent := g.GetAdjacent(x, y)
				emptySpaces := make([][2]int, 0)

				for _, pos := range adjacent {
					if g.grid[pos[1]][pos[0]].Type == Empty {
						if val, _ := moved.Get(pos); !val {
							emptySpaces = append(emptySpaces, pos)
						}
					}
				}

				if len(emptySpaces) > 0 {
					newPos := emptySpaces[rand.Intn(len(emptySpaces))]
					cell.BreedTime--

					if cell.BreedTime <= 0 {
						// Reproduce
						newGrid[y][x] = Cell{
							Type:      Fish,
							BreedTime: fishBreedTime,
						}
						cell.BreedTime = fishBreedTime
					}
					newGrid[newPos[1]][newPos[0]] = cell
					moved.Set(newPos, true)
				} else {
					newGrid[y][x] = cell
				}
			}
		}
	}
}

type SafeMap struct {
	mu    sync.Mutex
	moved map[[2]int]bool
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		moved: make(map[[2]int]bool),
	}
}

func (sm *SafeMap) Set(key [2]int, value bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.moved[key] = value
}

func (sm *SafeMap) Get(key [2]int) (bool, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	val, ok := sm.moved[key]
	return val, ok
}

// Update Update function to use SafeMap
func (g *Game) Update() error {
	newGrid := &[gridSize][gridSize]Cell{}
	safeMap := NewSafeMap()
	var wg sync.WaitGroup
	if time.Since(current).Milliseconds() > 500 {
		WriteTPS()
		current = time.Now()
	}

	for _, thread := range threads {
		wg.Add(2) // One for shark, one for fish
		go func(t ThreadBounds) {
			defer wg.Done()
			updateShark(newGrid, t.MinY, t.MaxY, safeMap, g)
		}(thread)
		go func(t ThreadBounds) {
			defer wg.Done()
			updateFish(newGrid, t.MinY, t.MaxY, safeMap, g)
		}(thread)
	}

	wg.Wait()
	g.grid = *newGrid
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	cellSize := 5 // or whatever size you want each cell to be

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

			// Draw a rectangle for each cell
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

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

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
