package pricing

import (
	"fmt"

	"pos-backend/internal/pos/domain/shared"
)

const BasisPointDenominator int64 = 10000

// RoundRatioHalfUp выполняет воспроизводимое целочисленное округление half-up.
func RoundRatioHalfUp(amount, numerator, denominator int64) (int64, error) {
	if amount < 0 || numerator < 0 || denominator <= 0 {
		return 0, fmt.Errorf("%w: invalid rounding ratio", shared.ErrInvalid)
	}
	product := amount * numerator
	if amount != 0 && product/amount != numerator {
		return 0, fmt.Errorf("%w: rounding ratio overflow", shared.ErrInvalid)
	}
	return (product + denominator/2) / denominator, nil
}

func percentOf(amount, basisPoints int64) (int64, error) {
	if basisPoints < 0 {
		return 0, fmt.Errorf("%w: percentage basis points must be non-negative", shared.ErrInvalid)
	}
	return RoundRatioHalfUp(amount, basisPoints, BasisPointDenominator)
}

func inclusiveTaxOf(gross, basisPoints int64) (int64, error) {
	if basisPoints < 0 {
		return 0, fmt.Errorf("%w: tax basis points must be non-negative", shared.ErrInvalid)
	}
	return RoundRatioHalfUp(gross, basisPoints, BasisPointDenominator+basisPoints)
}

func allocateProportionally(total int64, bases []int64) []int64 {
	out := make([]int64, len(bases))
	if total <= 0 || len(bases) == 0 {
		return out
	}
	var baseTotal int64
	for _, base := range bases {
		if base > 0 {
			baseTotal += base
		}
	}
	if baseTotal <= 0 {
		return out
	}
	type remainder struct {
		index int
		value int64
	}
	remainders := make([]remainder, 0, len(bases))
	var assigned int64
	for i, base := range bases {
		if base <= 0 {
			continue
		}
		product := total * base
		share := product / baseTotal
		out[i] = share
		assigned += share
		remainders = append(remainders, remainder{index: i, value: product % baseTotal})
	}
	for remaining := total - assigned; remaining > 0; remaining-- {
		best := -1
		for i, item := range remainders {
			if best < 0 || item.value > remainders[best].value || item.value == remainders[best].value && item.index < remainders[best].index {
				best = i
			}
		}
		if best < 0 {
			break
		}
		out[remainders[best].index]++
		remainders[best].value = -1
	}
	return out
}
