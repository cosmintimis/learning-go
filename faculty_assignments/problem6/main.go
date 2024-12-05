package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Graph represented as an adjacency list
type Graph struct {
	adjacencyList map[int][]int
}

func ReadGraphFromFile(filename string) (Graph, error) {
	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		return Graph{}, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Read the file content
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return Graph{}, fmt.Errorf("error reading file: %w", err)
	}

	// Parse the JSON into the adjacency list
	adjacencyList := make(map[int][]int)
	err = json.Unmarshal(byteValue, &adjacencyList)
	if err != nil {
		return Graph{}, fmt.Errorf("error parsing JSON: %w", err)
	}

	// Return the graph instance
	return Graph{adjacencyList: adjacencyList}, nil
}

func findHamiltonianCycleParallelized(graph Graph, startVertex int) {
	numVertices := len(graph.adjacencyList)
	resultChan := make(chan []int)
	defer close(resultChan)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	path := make([]int, numVertices)
	visited := make(map[int]bool)
	for i := 0; i < numVertices; i++ {
		path[i] = -1
	}
	path[0] = startVertex
	visited[startVertex] = true

	wg.Add(1)
	go search(graph, numVertices, 1, path, visited, resultChan, ctx, &wg)

	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case cycle := <-resultChan:
		if cycle != nil {
			fmt.Println("Hamiltonian Cycle Found:", cycle)
		} else {
			fmt.Println("No Hamiltonian Cycle Found")
		}
	case <-ctx.Done():
		fmt.Println("Search canceled or completed without a cycle")
	}
}

func search(graph Graph, numVertices int, pos int, path []int, visited map[int]bool, resultChan chan []int, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return
	default:
	}
	// If all vertices are visited and the last vertex is adjacent to the start vertex, we have found a Hamiltonian cycle
	if pos == numVertices {
		if contains(graph.adjacencyList[path[pos-1]], path[0]) {
			// add the start vertex to the end of the path
			path = append(path, path[0])
			select {
			case resultChan <- append([]int(nil), path...):
			case <-ctx.Done():
			}
		}
		return
	}

	for v := 0; v < numVertices; v++ {
		if isValidV2(graph, v, pos, path, visited) {
			// Clone visited map to avoid concurrent modifications
			visitedClone := make(map[int]bool)
			for key, val := range visited {
				visitedClone[key] = val
			}
			visitedClone[v] = true
			newPath := make([]int, numVertices)
			copy(newPath, path)
			newPath[pos] = v
			wg.Add(1)
			go search(graph, numVertices, pos+1, newPath, visitedClone, resultChan, ctx, wg)
		}
	}

}

func isValidV2(graph Graph, v int, pos int, path []int, visited map[int]bool) bool {
	// Check if this vertex is adjacent to the previous vertex in the path
	if !contains(graph.adjacencyList[path[pos-1]], v) {
		return false
	}
	// Check if the vertex has already been visited
	if visited[v] {
		return false
	}
	return true
}

func findHamiltonianCycleClassic(graph Graph, startVertex int) []int {
	vertices := len(graph.adjacencyList)
	path := make([]int, vertices)
	for i := 0; i < vertices; i++ {
		path[i] = -1
	}
	path[0] = startVertex
	if !solveHamiltonian(graph, path, 1) {
		return nil
	}
	// Add the start vertex to the end of the path
	path = append(path, startVertex)
	return path
}

func contains(slice []int, element int) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

func solveHamiltonian(graph Graph, path []int, pos int) bool {
	if pos == len(path) {
		// check if the last vertex is connected to the first vertex
		return path[pos-1] != path[0] && contains(graph.adjacencyList[path[pos-1]], path[0])
	}
	for v := 0; v < len(path); v++ {
		if isValid(graph, v, pos, path) {
			path[pos] = v
			if solveHamiltonian(graph, path, pos+1) {
				return true
			}
			path[pos] = -1 // Backtracking
		}
	}
	return false
}

func isValid(graph Graph, v int, pos int, path []int) bool {
	// check if this vertex is an adjacent vertex of the previously added vertex.
	if !contains(graph.adjacencyList[path[pos-1]], v) {
		return false
	}
	for i := 0; i < pos; i++ {
		if path[i] == v { // check if the vertex has already been included.
			return false
		}
	}
	return true
}

func main() {
	// graph := Graph{
	// 	adjacencyList: map[int][]int{
	// 		0: {2},
	// 		1: {0},
	// 		2: {3},
	// 		3: {1},
	// 	},
	// }
	// res := findHamiltonianCycleClassic(graph, 0)
	// fmt.Println("Hamiltonian Cycle Found:", res)
	// findHamiltonianCycleParallelized(graph, 0)
	graph, err := ReadGraphFromFile("graph_50_200.json")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	start := time.Now()
	res := findHamiltonianCycleClassic(graph, 0)
	fmt.Println("Classic:", time.Since(start))
	fmt.Println("Hamiltonian Cycle Found:", res)
	start = time.Now()
	findHamiltonianCycleParallelized(graph, 0)
	fmt.Println("Parallelized:", time.Since(start))

}
