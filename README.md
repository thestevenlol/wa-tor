# Wa-Tor Simulation written in Go | Jack Foley - C00274246

**GitHub Link**
---------------
https://github.com/thestevenlol/wa-tor

**Introduction**
---------------

This Go program simulates the Wator game, which is a simple simulation of an ecosystem where fish and sharks live together and interact. The simulation is designed to run concurrently using multiple threads, where each thread will be responsible for updating part of the game grid.

**Game Summary**
-----------------

The Wator game is a simple simulation in a grid where fish and sharks interact with each other. The rules of the game are as follows:

Fish move and spawn after a fixed number of moves.
Sharks swim around and eat fish to survive. If a shark does not eat a fish after a certain number of steps, it will die.
Blank spaces on the grid may be occupied by either fish or sharks.

**Code Structure**
---------------

The code is organised in several packages and files:

* `package wator`: The main package containing the Wator simulation code.
* `Game struct`: Stores the state of the game, including the grid and simulation parameters.
* `ThreadGrid struct`: A subset of the grid assigned to a thread.
* `Cell struct`: Represents a single cell in the grid. Contains the type of cell (fish, shark, or empty) along with simulation information.

**Simulation Loop**
--------------------

The simulation loop is used in the `Update` function of the `Game` struct. The loop has these steps:

1. **Initialise Occupied Grid**: Set the occupied grid to zero, meaning no cells are occupied by threads.
2. **Create Thread Grids**: Build a `ThreadGrid` struct for every thread, and assign each thread a section of the grid.
3. **Update Shark Cells**: Update the shark cells in the grid by randomly moving them and eating fish to survive.
4. **Update Fish Cells**: Update the fish cells in the grid by moving them in random directions and having them reproduce after a certain number of steps.
5. **Refresh Grid**: Update the game grid to reflect the updated cell values.

**Concurrency**
------------------

The simulation uses many threads at the same time. Each thread updates a part of the grid. The `ThreadGrid` struct represents a part of the grid given to a thread, and the `Update` function changes the cells in that part of the grid.

**TPS Measurement**
------------------

The simulation has a feature to measure TPS (ticks per second) and write it to a CSV file. The `WriteTPS` function is called at regular intervals to update the TPS measurement.

**Running the Simulation**
-------------------------

To run the simulation, just run the `main` function. The simulation will run without stopping, updating the grid and showing the current state on the screen. The measurement of TPS will be written to a CSV file from time to time.

**Requirements** 
------------

To run the simulation, you will need to have Go installed on your computer. It uses the Ebiten library for displaying graphics; you can install it with this command:

```bash 
go get -u github.com/hajimehoshi/ebiten/v2 
``` 

**Example Use Case** 
------------------ 
Running the simulation in 16 threads is as simple as calling the `main` function: 

```bash 
go run main.go 
``` 

This will activate the simulation and show the actual state in the screen. The TPS measurement will be written to a CSV file every 500ms.

**Documentation**
------------------ 

To view the documentation for the project, in the root folder you will find a file called `wator - Go Documentation Server.htm`. You can open this file with your favourite search engine and view it online. 