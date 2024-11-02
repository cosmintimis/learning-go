# Problem 3: Simple parallel tasks

## Overview

Divide a simple task between threads. The task can easily be divided in sub-tasks requiring no cooperation at all. See the caching effects, and the costs of creating threads and of switching between threads.

## Requirement

Write several programs to compute the product of two matrices.

Have a function that computes a single element of the resulting matrix.

Have a second function whose each call will constitute a parallel task (that is, this function will be called on several threads in parallel). This function will call the above one several times consecutively to compute several elements of the resulting matrix.

For running the tasks, also implement 2 approaches:

1. Create an actual thread for each task (use the low-level thread mechanism from the programming language);
2. Use a thread pool.

Experiment with various values for (and document the attempts and their performance):

- The sizes of the matrix;
- The number of tasks (this is equal to the number of threads when not using a thread pool);
- The number of threads and other parameters for the thread pool (when using the thread pool).
