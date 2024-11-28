package main

import (
	"fmt"
	"image/color"
	"log"
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
	nThreads        = 12
	subGridSize     = gridSize / nThreads
)

type CellType int

const (
	Empty CellType = iota
	Fish
	Shark
)

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

func getBlockSize(threadId int) [][2]int {
	blockSize := gridSize / nThreads
	startX := threadId * blockSize
	endX := startX + blockSize
	startY := threadId * blockSize
	endY := startY + blockSize

	if threadId == nThreads-1 {
		endX = gridSize
		endY = gridSize
	}

	return [][2]int{{startX, endX}, {startY, endY}}
}

func assignThreadBlocks() map[int][][2]int {
	threads := make(map[int][][2]int) // threadId -> [[startX,endX],[startY,endY]]
	for i := 0; i < nThreads; i++ {
		threads[i] = getBlockSize(i)
	}
	return threads
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

func updateShark(newGrid *[gridSize][gridSize]Cell, moved map[[2]int]bool, g *Game) error {
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Shark {
				adjacent := g.GetAdjacent(x, y)
				Shuffle(adjacent)

				// Look for fish to eat
				fishFound := false
				for _, pos := range adjacent {
					if g.grid[pos[1]][pos[0]].Type == Fish && !moved[pos] {
						// Shark eats fish and moves
						newGrid[pos[1]][pos[0]] = Cell{
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
						moved[newPos] = true
					} else {
						newGrid[y][x] = cell
					}
				}
			}
		}
	}
	return nil
}

func updateFish(newGrid *[gridSize][gridSize]Cell, moved map[[2]int]bool, g *Game) {
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Fish {
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
						newGrid[y][x] = Cell{
							Type:      Fish,
							BreedTime: fishBreedTime,
						}
						cell.BreedTime = fishBreedTime
					}
					newGrid[newPos[1]][newPos[0]] = cell
					moved[newPos] = true
				} else {
					newGrid[y][x] = cell
				}
			}
		}
	}
}

// Update function
// Parameters: None
// Returns: error
// Description: Updates the game state by simulating one step of the Wa-Tor world simulation.
func (g *Game) Update() error {
	// Use pointer to grid
	newGrid := &[gridSize][gridSize]Cell{}
	moved := make(map[[2]int]bool)

	updateShark(newGrid, moved, g)
	updateFish(newGrid, moved, g)

	// Copy new grid state
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

func main() {
	threadLimits := assignThreadBlocks()
	for threadId, limits := range threadLimits {
		fmt.Printf("Thread %d: X[%d:%d] Y[%d:%d]\n",
			threadId,
			limits[0][0], limits[0][1],
			limits[1][0], limits[1][1])
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
