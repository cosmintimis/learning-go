package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/proabiral/puredns/v2/pkg/threadpool"
)

// used for the thread pool
type MyTask struct {
	start  int
	end    int
	poly1  *Polynomial
	poly2  *Polynomial
	result *Polynomial
}

func (m *MyTask) Run() {
	for i := m.start; i < m.end; i++ {
		if i > m.result.degree+1 {
			break
		}
		for j := 0; j <= i; j++ {
			if j > m.poly1.degree || i-j > m.poly2.degree {
				continue
			}
			m.result.coefficients[i] += m.poly1.coefficients[j] * m.poly2.coefficients[i-j]
		}
	}
}

type Polynomial struct {
	coefficients []float64
	degree       int
}

func (p *Polynomial) add(q *Polynomial) *Polynomial {
	maxDegree := max(p.degree, q.degree)
	result := Polynomial{make([]float64, maxDegree+1), maxDegree}
	for i := 0; i <= maxDegree; i++ {
		if i <= p.degree {
			result.coefficients[i] += p.coefficients[i]
		}
		if i <= q.degree {
			result.coefficients[i] += q.coefficients[i]
		}
	}
	// remove leading zeros
	for i := maxDegree; i >= 0; i-- {
		if result.coefficients[i] != 0 {
			result.degree = i
			break
		}
	}
	result.coefficients = result.coefficients[:result.degree+1]
	return &result
}

func (p *Polynomial) multiply(q *Polynomial) *Polynomial {
	result := Polynomial{make([]float64, p.degree+q.degree+1), p.degree + q.degree}
	for i := 0; i <= p.degree; i++ {
		for j := 0; j <= q.degree; j++ {
			result.coefficients[i+j] += p.coefficients[i] * q.coefficients[j]
		}
	}
	return &result
}

func (p *Polynomial) multiplyParallelized(q *Polynomial, step int) Polynomial {
	result := Polynomial{make([]float64, p.degree+q.degree+1), p.degree + q.degree}
	myThreadPool := threadpool.NewThreadPool(step, 1000)
	for i := 0; i < result.degree+1; i += step {
		myTask := &MyTask{
			start:  i,
			end:    i + step,
			poly1:  p,
			poly2:  q,
			result: &result,
		}
		myThreadPool.Execute(myTask)
	}
	myThreadPool.Wait()
	myThreadPool.Close()
	return result
}

func (p *Polynomial) karatsuba(q *Polynomial) *Polynomial {
	if p.degree < 100 || q.degree < 100 {
		return p.multiply(q)
	}
	length := min(p.degree, q.degree) + 1
	// split the polynomials
	low1 := Polynomial{p.coefficients[:length/2], length/2 - 1}
	high1 := Polynomial{p.coefficients[length/2:], length/2 - 1}
	low2 := Polynomial{q.coefficients[:length/2], length/2 - 1}
	high2 := Polynomial{q.coefficients[length/2:], length/2 - 1}
	// calculate the three products
	z0 := low1.karatsuba(&low2)
	z1 := low1.add(&high1).karatsuba(low2.add(&high2))
	z2 := high1.karatsuba(&high2)
	// calculate the result
	result := z2.shift(2 * length).add(z1.add(z2.negate().add(z0.negate()))).shift(length).add(z0)
	return result
}

func (p *Polynomial) karatsubaParallelized(q *Polynomial) *Polynomial {
	var wg sync.WaitGroup
	if p.degree < 100 || q.degree < 100 {
		return p.multiply(q)
	}
	length := min(p.degree, q.degree) + 1
	// split the polynomials
	low1 := Polynomial{p.coefficients[:length/2], length/2 - 1}
	high1 := Polynomial{p.coefficients[length/2:], length/2 - 1}
	low2 := Polynomial{q.coefficients[:length/2], length/2 - 1}
	high2 := Polynomial{q.coefficients[length/2:], length/2 - 1}
	// calculate the three products
	var z0, z1, z2 *Polynomial
	wg.Add(1)
	go func() {
		z0 = low1.karatsubaParallelized(&low2)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		z1 = low1.add(&high1).karatsubaParallelized(low2.add(&high2))
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		z2 = high1.karatsubaParallelized(&high2)
		wg.Done()
	}()
	wg.Wait()
	// calculate the result
	result := z2.shift(2 * length).add(z1.add(z2.negate().add(z0.negate()))).shift(length).add(z0)
	return result
}

func (p *Polynomial) negate() *Polynomial {
	result := Polynomial{make([]float64, p.degree+1), p.degree}
	for i := 0; i <= p.degree; i++ {
		result.coefficients[i] = -p.coefficients[i]
	}
	return &result
}

func (p *Polynomial) shift(shift int) *Polynomial {
	result := Polynomial{make([]float64, p.degree+shift+1), p.degree + shift}
	for i := 0; i <= p.degree; i++ {
		result.coefficients[i+shift] = p.coefficients[i]
	}
	return &result
}

func (p *Polynomial) toString() string {
	result := ""
	for i := 0; i <= p.degree; i++ {
		if p.coefficients[i] != 0 {
			if p.coefficients[i] > 0 && i != 0 {
				result += "+"
			}
			result += fmt.Sprintf("%.2fx^%d", p.coefficients[i], i)
		}
	}
	return result
}

func generatePolynomial(degree int) Polynomial {
	coefficients := make([]float64, degree+1)
	for i := 0; i <= degree; i++ {
		if rand.Float64() < 0.5 {
			coefficients[i] = rand.Float64() * 100
		} else {
			coefficients[i] = rand.Float64() * -100
		}
	}
	return Polynomial{coefficients, degree}
}

func main() {
	// p := Polynomial{[]float64{1, 2, 3}, 2}
	// q := Polynomial{[]float64{4, 5}, 1}
	// fmt.Println(p.toString())
	// fmt.Println(q.toString())
	// r1 := p.multiply(&q)
	// r2 := p.multiplyParallelized(&q, 2)
	// fmt.Println(r1.toString())
	// fmt.Println(r2.toString())
	// r3 := p.karatsuba(&q)
	// fmt.Println(r3.toString())
	// r4 := p.karatsubaParallelized(&q)
	// fmt.Println(r4.toString())

	p := generatePolynomial(10000)
	q := generatePolynomial(10000)
	start := time.Now()
	p.multiply(&q)
	elapsed := time.Since(start)
	fmt.Printf("Multiplication took %s\n", elapsed)
	start = time.Now()
	p.multiplyParallelized(&q, 100)
	elapsed = time.Since(start)
	fmt.Printf("Parallelized multiplication took %s\n", elapsed)
	start = time.Now()
	p.karatsuba(&q)
	elapsed = time.Since(start)
	fmt.Printf("Karatsuba took %s\n", elapsed)
	start = time.Now()
	p.karatsubaParallelized(&q)
	elapsed = time.Since(start)
	fmt.Printf("Parallelized Karatsuba took %s\n", elapsed)
}
