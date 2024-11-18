package main

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/exp/rand"
)

const (
	screenWidth     = 720
	screenHeight    = 720
	gridSize        = 60
	cellSize        = screenWidth / gridSize
	fishBreedTime   = 3
	sharkBreedTime  = 8
	sharkStarveTime = 3
	fishPercentage  = 50
	sharkPercentage = 20
)

type CellType int

const (
	Empty CellType = iota
	Fish
	Shark
)

type Cell struct {
	Type       CellType
	BreedTime  int
	StarveTime int
}

type Game struct {
	grid [gridSize][gridSize]Cell
}

/**
 * @brief Initialises the game grid with fish, sharks, and empty cells based on predefined probabilities.
 *
 * This method populates the game grid with cells containing fish, sharks, or empty spaces.
 * The probabilities for each cell type are determined by the constants `fishPercentage` and `sharkPercentage`.
 * Each cell containing a fish or shark is initialised with its respective breed and starvation times.
 *
 * @details The grid is a 2D array, and each cell is assigned one of the following types:
 * - Fish: Assigned if a random number is less than `fishPercentage`.
 * - Shark: Assigned if a random number falls between `fishPercentage` and `fishPercentage + sharkPercentage`.
 * - Empty: Assigned otherwise.
 *
 * The function uses the following constants:
 * - `fishPercentage`: The percentage of cells that should initially contain fish.
 * - `sharkPercentage`: The percentage of cells that should initially contain sharks.
 * - `fishBreedTime`: The time for fish to reproduce.
 * - `sharkBreedTime`: The time for sharks to reproduce.
 * - `sharkStarveTime`: The time for sharks to starve without food.
 *
 * @note Assumes `gridSize` is a square dimension of the grid and `rand.Intn` generates a random integer.
 *
 * @param None This method does not take any parameters as it operates directly on the `Game` object.
 * @return void This method does not return a value.
 */
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

func (g *Game) getAdjacent(x, y int) [][2]int {
	adjacent := make([][2]int, 0, 4)
	directions := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	for _, d := range directions {
		newX := (x + d[0] + gridSize) % gridSize
		newY := (y + d[1] + gridSize) % gridSize
		adjacent = append(adjacent, [2]int{newX, newY})
	}
	return adjacent
}

func (g *Game) Update() error {
	// Create temporary grid to store next state
	newGrid := [gridSize][gridSize]Cell{}
	moved := make(map[[2]int]bool)

	// First pass: Update sharks
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Shark {
				adjacent := g.getAdjacent(x, y)

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

	// Second pass: Update fish
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if moved[[2]int{x, y}] {
				continue
			}

			cell := g.grid[y][x]
			if cell.Type == Fish {
				adjacent := g.getAdjacent(x, y)
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

	// Copy new grid state
	g.grid = newGrid
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
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	rand.Seed(uint64(time.Now().UnixNano()))
	game := &Game{}
	game.Initialise()
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Wator Simulation in Go! (Ebiten)")
	ebiten.SetTPS(10)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
