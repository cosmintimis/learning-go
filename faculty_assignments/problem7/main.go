package main

import (
	"fmt"
	"time"

	mpi "github.com/mnlphlp/gompi"
	"github.com/mnlphlp/gompi/comm"
)

func sequentialMultiplyWorker(comm comm.Communicator) {
	var pCoefficients, qCoefficients, rCoefficients []float64
	var pDegree, qDegree, rDegree, start, end int
	comm.Recv(&pCoefficients, 0, 0)
	comm.Recv(&pDegree, 0, 0)
	comm.Recv(&qCoefficients, 0, 0)
	comm.Recv(&qDegree, 0, 0)
	comm.Recv(&rCoefficients, 0, 0)
	comm.Recv(&rDegree, 0, 0)
	comm.Recv(&start, 0, 0)
	comm.Recv(&end, 0, 0)

	// fmt.Println("Worker: ", comm.GetRank(), " start: ", start, " end: ", end)

	for i := start; i < end; i++ {
		if i > rDegree+1 {
			break
		}
		for j := 0; j <= i; j++ {
			if j > pDegree || i-j > qDegree {
				continue
			}
			rCoefficients[i] += pCoefficients[j] * qCoefficients[i-j]
		}
	}

	// send the result back to the master process
	comm.Send(rCoefficients, 0, 0)
}

func sequentialMultiplyMaster(processesNumber int, comm comm.Communicator) {
	// Master process
	// Generate two polynomials
	p := generatePolynomial(10000)
	q := generatePolynomial(10000)
	r := createEmptyPolynomial(p.degree + q.degree)

	step := (r.degree + 1) / (processesNumber - 1)
	step += 2 // add a tolerance of 2
	// fmt.Println("Step: ", step)
	var start, end int
	// Distribute the work
	startTime := time.Now()
	for i := 1; i < processesNumber; i++ {
		start = end
		end = start + step

		// fmt.Println("Master: ", i, " start: ", start, " end: ", end)

		pDegree := p.degree
		qDegree := q.degree
		rDegree := r.degree
		comm.Send(p.coefficients, i, 0)
		comm.Send(&pDegree, i, 0)
		comm.Send(q.coefficients, i, 0)
		comm.Send(&qDegree, i, 0)
		comm.Send(r.coefficients, i, 0)
		comm.Send(&rDegree, i, 0)
		comm.Send(&start, i, 0)
		comm.Send(&end, i, 0)
	}

	// Collect the results
	for i := 1; i < processesNumber; i++ {
		var resultedCoefficients []float64
		comm.Recv(&resultedCoefficients, i, 0)

		// Update the result
		for j := 0; j < r.degree+1; j++ {
			r.coefficients[j] += resultedCoefficients[j]
		}
	}
	endTime := time.Since(startTime)

	// Print the time
	fmt.Println("Sequential multiplication took: ", endTime)

	// Print the result
	// fmt.Println("Result: ", r.toString())
}

func karatsubaMultiplyWorker(comm comm.Communicator) {
	var pCoefficients, qCoefficients, rCoefficients []float64
	var rDegree, start, end int
	comm.Recv(&pCoefficients, 0, 0)
	comm.Recv(&qCoefficients, 0, 0)
	comm.Recv(&rCoefficients, 0, 0)
	comm.Recv(&rDegree, 0, 0)
	comm.Recv(&start, 0, 0)
	comm.Recv(&end, 0, 0)

	if start >= rDegree {
		return
	}

	if end > rDegree {
		end = rDegree
	}

	// fmt.Println("Worker: ", comm.GetRank(), " start: ", start, " end: ", end)

	for i := start; i < end; i++ {
		if i%2 == 0 {
			rCoefficients[i] = karatsubaUtilEvenValue(pCoefficients, qCoefficients, i)
		} else {
			rCoefficients[i] = karatsubaUtilOddValue(pCoefficients, qCoefficients, i)
		}
	}

	// fmt.Println("Worker: ", comm.GetRank(), " result: ", rCoefficients)

	// send the result back to the master process
	comm.Send(rCoefficients, 0, 0)
}

func karatsubaMultiplyMaster(processesNumber int, comm comm.Communicator) {
	// Master process
	// Generate two polynomials
	p := generatePolynomial(10000)
	q := generatePolynomial(10000)
	// p := Polynomial{[]float64{1, 2, 3, 4, 5}, 4}
	// q := Polynomial{[]float64{5, 4, 3, 2, 1}, 4}
	r := createEmptyPolynomial(p.degree + q.degree)
	r.coefficients[0] = p.coefficients[0] * q.coefficients[0]
	r.coefficients[r.degree] = p.coefficients[p.degree] * q.coefficients[q.degree]
	var start, end int
	end = 1

	// Distribute the work
	step := (r.degree - 1) / (processesNumber - 1)
	remainder := (r.degree - 1) % (processesNumber - 1)

	startTime := time.Now()
	for i := 1; i < processesNumber; i++ {
		start = end
		end = start + step

		if remainder > 0 {
			end++
			remainder--
		}

		rDegree := r.degree
		comm.Send(p.coefficients, i, 0)
		comm.Send(q.coefficients, i, 0)
		comm.Send(r.coefficients, i, 0)
		comm.Send(&rDegree, i, 0)
		comm.Send(&start, i, 0)
		comm.Send(&end, i, 0)
	}

	// Collect the results
	for i := 1; i < processesNumber; i++ {
		var resultedCoefficients []float64
		comm.Recv(&resultedCoefficients, i, 0)

		// Update the result
		for j := 1; j < r.degree; j++ {
			r.coefficients[j] += resultedCoefficients[j]
		}
	}
	endTime := time.Since(startTime)

	// Print the time
	fmt.Println("Karatsuba multiplication took: ", endTime)

	// Print the result
	//fmt.Println("Result: ", r.toString())

}

type IMPLEMENTATION_TYPE string

const (
	SEQUENTIAL IMPLEMENTATION_TYPE = "SEQUENTIAL"
	KARATSUBA  IMPLEMENTATION_TYPE = "KARATSUBA"
)

func main() {
	IMPLEMENTATION_TYPE := KARATSUBA
	mpi.Init()
	defer mpi.Finalize()

	comm := mpi.NewComm(false)
	processesNumber := comm.GetSize()
	rank := comm.GetRank()
	fmt.Println("Process with rank ", rank, " started")
	// sequential multiplication of two polynomials
	if rank == 0 {
		if IMPLEMENTATION_TYPE == SEQUENTIAL {
			sequentialMultiplyMaster(processesNumber, comm)
		} else {
			karatsubaMultiplyMaster(processesNumber, comm)
		}

	} else {
		if IMPLEMENTATION_TYPE == SEQUENTIAL {
			sequentialMultiplyWorker(comm)
		} else {
			karatsubaMultiplyWorker(comm)
		}
	}
}
