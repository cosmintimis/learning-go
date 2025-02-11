# Problem 6: Parallelizing techniques (2 - parallel explore)

## Overview

The goal of this lab is to implement a simple but non-trivial parallel algorithm.

## Requirement

Given a directed graph, find a Hamiltonean cycle, if one exists. Use multiple threads to parallelize the search. Important The search should start from a fixed vertex (no need to take each vertex as the starting point), however, the splitting of the work between threads should happen at several levels, for all possible choices among the neighbors of each current vertex.

The documentation will describe:
- the algorithms
- the synchronization used in the parallelized variants
- the performance measurements

### Bonus: do the same for big numbers.