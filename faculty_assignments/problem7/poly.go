package main

import (
	"fmt"
	"math/rand"
)

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

func createEmptyPolynomial(degree int) Polynomial {
	coefficients := make([]float64, degree+1)
	return Polynomial{coefficients, degree}
}

/*
Di = ai * bi  [∀ i = 0, 1, 2, ..., n-1]

Dp,q = (ap + aq) * (bp + bq)  [∀ i = 1, 2, ..., 2n-3, and ∀ p, q such that p + q = i and q > p ≥ 0]

	ci = {
		Σ Dp,q - Σ (Dp + Dq), for odd values of i, 0 < i < 2n - 2
		Σ Dp + Dq + Di/2, for even values of i, 0 < i < 2n - 2
	}
*/

func karatsubaUtilOddValue(pCoeff []float64, qCoeff []float64, i int) float64 {
	sum1 := 0.0
	sum2 := 0.0

	for p := 0; p <= i; p++ {
		q := i - p
		if q <= p { // Ensure q > p
			break
		}

		// Preload values to avoid repeated function calls
		pCoeffP := getValueOfArray(pCoeff, p)
		pCoeffQ := getValueOfArray(pCoeff, q)
		qCoeffP := getValueOfArray(qCoeff, p)
		qCoeffQ := getValueOfArray(qCoeff, q)

		// Compute sum1 and sum2
		sum1 += (pCoeffP + pCoeffQ) * (qCoeffP + qCoeffQ)
		sum2 += (pCoeffP * qCoeffP) + (pCoeffQ * qCoeffQ)
	}

	return sum1 - sum2
}

func karatsubaUtilEvenValue(pCoeff []float64, qCoeff []float64, i int) float64 {
	return karatsubaUtilOddValue(pCoeff, qCoeff, i) + getValueOfArray(pCoeff, i/2)*getValueOfArray(qCoeff, i/2)
}

func getValueOfArray(arr []float64, index int) float64 {
	if index < 0 || index >= len(arr) {
		return 0
	}
	return arr[index]
}
