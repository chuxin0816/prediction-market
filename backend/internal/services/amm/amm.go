package amm

import (
	"encoding/json"
	"errors"

	"github.com/prediction-market/backend/internal/models"
	"github.com/shopspring/decimal"
)

// ParseReserves extracts decimal reserves from a LiquidityPool's JSON field.
func ParseReserves(pool *models.LiquidityPool) ([]decimal.Decimal, error) {
	var raw []string
	if err := json.Unmarshal(pool.OutcomeReserves, &raw); err != nil {
		return nil, errors.New("failed to parse outcome reserves")
	}
	reserves := make([]decimal.Decimal, len(raw))
	for i, s := range raw {
		d, err := decimal.NewFromString(s)
		if err != nil {
			return nil, errors.New("invalid reserve value")
		}
		reserves[i] = d
	}
	return reserves, nil
}

// MarshalReserves converts decimal reserves back to JSON for storage.
func MarshalReserves(reserves []decimal.Decimal) ([]byte, error) {
	raw := make([]string, len(reserves))
	for i, r := range reserves {
		raw[i] = r.String()
	}
	return json.Marshal(raw)
}

// product computes the product of all elements in the slice.
func product(vals []decimal.Decimal) decimal.Decimal {
	p := decimal.NewFromInt(1)
	for _, v := range vals {
		p = p.Mul(v)
	}
	return p
}

// CalculateBuy computes the number of outcome shares a user receives when
// spending usdcAmount on outcomeIndex.
//
// Algorithm (CPMM):
//  1. Record old k = product of all reserves.
//  2. Mint usdcAmount complete sets: add usdcAmount to EVERY reserve.
//  3. Remove the desired outcome tokens from the pool; we need to figure out
//     how many the pool can release while keeping the invariant.
//     - New reserves for all j != outcomeIndex are (Rj + usdcAmount).
//     - Ri_new = k / product(all new Rj where j != i).
//  4. sharesOut = (Ri_old + usdcAmount) - Ri_new
func CalculateBuy(pool *models.LiquidityPool, outcomeIndex int, usdcAmount decimal.Decimal) (sharesOut decimal.Decimal, newReserves []decimal.Decimal, err error) {
	reserves, err := ParseReserves(pool)
	if err != nil {
		return decimal.Zero, nil, err
	}

	if outcomeIndex < 0 || outcomeIndex >= len(reserves) {
		return decimal.Zero, nil, errors.New("invalid outcome index")
	}
	if usdcAmount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil, errors.New("amount must be positive")
	}

	// k = product of current reserves
	k := product(reserves)

	// After minting complete sets, every reserve increases by usdcAmount.
	// But we only keep the increase for reserves j != outcomeIndex in the pool;
	// the outcome-i tokens are what the user wants.
	newReserves = make([]decimal.Decimal, len(reserves))
	for j := range reserves {
		newReserves[j] = reserves[j].Add(usdcAmount)
	}

	// Compute product of all new reserves except outcomeIndex.
	prodOthers := decimal.NewFromInt(1)
	for j, r := range newReserves {
		if j != outcomeIndex {
			prodOthers = prodOthers.Mul(r)
		}
	}

	// Ri_new = k / prodOthers
	if prodOthers.IsZero() {
		return decimal.Zero, nil, errors.New("division by zero in AMM calculation")
	}
	riNew := k.Div(prodOthers)

	// sharesOut = (Ri_old + usdcAmount) - Ri_new
	sharesOut = newReserves[outcomeIndex].Sub(riNew)
	if sharesOut.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil, errors.New("calculated shares out is non-positive")
	}

	newReserves[outcomeIndex] = riNew
	return sharesOut, newReserves, nil
}

// CalculateSell computes the USDC a user receives when selling sharesIn of
// outcomeIndex back to the pool.
//
// Algorithm (reverse of buy):
//  1. Record old k = product of all reserves.
//  2. Add sharesIn back to reserve i: Ri_new = Ri + sharesIn.
//  3. For each j != i, compute Rj_new = k / product(all reserves except j, using new Ri).
//     But since we need to find how many complete sets can be removed, we use:
//     - prodOthersWithNewI = product of all reserves except j, but with Ri replaced by Ri_new
//     Actually a simpler formulation:
//     - After adding shares back: new_Ri = Ri + sharesIn
//     - k must be maintained, so product of new reserves = k
//     - new k_with_extra = Ri_new * product(all Rj for j != i) > k
//     - We need to remove tokens from other reserves to bring back to k
//     - For each j != i: Rj_new = k / product(all reserves except j, with Ri = Ri_new)
//     - But all Rj are the same, so: prodExceptJ for each j is the same only in 2-outcome.
//     - General: compute new product with Ri_new, figure out scaling.
//
// Simpler approach: after adding sharesIn to reserve i, we have overcapacity.
// We need to reduce all OTHER reserves proportionally to restore k.
// Let newRi = Ri + sharesIn. Then product of other reserves must equal k / newRi.
// Current product of others = k / Ri.
// So ratio = (k / newRi) / (k / Ri) = Ri / newRi.
// Each other reserve scales by ratio^(1/(n-1))? No, that's not right for CPMM.
//
// Actually the correct approach for selling in CPMM prediction markets:
//  1. Add sharesIn to reserve i.
//  2. Now the pool has excess value. We want to extract complete sets.
//  3. Let d = amount of complete sets removed (= USDC out).
//  4. New reserves: Ri' = Ri + sharesIn - d, Rj' = Rj - d for j != i.
//  5. Constraint: product of all new reserves = k.
//  6. For 2 outcomes: (R1 + S - d)(R2 - d) = k, solve for d.
//  7. For n outcomes this is a polynomial of degree n. We use Newton's method.
func CalculateSell(pool *models.LiquidityPool, outcomeIndex int, sharesIn decimal.Decimal) (usdcOut decimal.Decimal, newReserves []decimal.Decimal, err error) {
	reserves, err := ParseReserves(pool)
	if err != nil {
		return decimal.Zero, nil, err
	}

	if outcomeIndex < 0 || outcomeIndex >= len(reserves) {
		return decimal.Zero, nil, errors.New("invalid outcome index")
	}
	if sharesIn.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil, errors.New("shares must be positive")
	}

	k := product(reserves)
	n := len(reserves)

	// For 2 outcomes, solve quadratically: (R_i + S - d)(R_j - d) = k
	if n == 2 {
		other := 1 - outcomeIndex
		ri := reserves[outcomeIndex].Add(sharesIn)
		rj := reserves[other]
		// (ri - d)(rj - d) = k
		// d^2 - (ri + rj)*d + (ri*rj - k) = 0
		a := decimal.NewFromInt(1)
		b := ri.Add(rj).Neg()                   // -(ri + rj)
		cCoeff := ri.Mul(rj).Sub(k)             // ri*rj - k
		discriminant := b.Mul(b).Sub(a.Mul(cCoeff).Mul(decimal.NewFromInt(4)))
		if discriminant.LessThan(decimal.Zero) {
			return decimal.Zero, nil, errors.New("no valid solution for sell")
		}
		// sqrt via Newton's method on decimal
		sqrtDisc := decimalSqrt(discriminant)
		// d = (-b - sqrt(disc)) / 2  (we want the smaller root)
		d := b.Neg().Sub(sqrtDisc).Div(decimal.NewFromInt(2))
		if d.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, nil, errors.New("calculated USDC out is non-positive")
		}
		// Verify d < rj (cannot drain a reserve)
		if d.GreaterThanOrEqual(rj) {
			return decimal.Zero, nil, errors.New("sell amount too large, would drain pool")
		}
		newReserves = make([]decimal.Decimal, 2)
		newReserves[outcomeIndex] = ri.Sub(d)
		newReserves[other] = rj.Sub(d)
		return d, newReserves, nil
	}

	// General case: Newton's method for n outcomes.
	// f(d) = product(Ri + Si - d for i=outcome, Rj - d for j!=outcome) - k = 0
	// where Si = sharesIn for the sold outcome, 0 otherwise.
	// We need to find d such that f(d) = 0.
	adjustedReserves := make([]decimal.Decimal, n)
	copy(adjustedReserves, reserves)
	adjustedReserves[outcomeIndex] = adjustedReserves[outcomeIndex].Add(sharesIn)

	// Upper bound for d: min of all adjustedReserves (cannot go negative)
	minReserve := adjustedReserves[0]
	for _, r := range adjustedReserves[1:] {
		if r.LessThan(minReserve) {
			minReserve = r
		}
	}

	// Newton's method
	d := minReserve.Div(decimal.NewFromInt(2)) // initial guess
	for iter := 0; iter < 100; iter++ {
		// f(d) = product(adjustedReserves[i] - d) - k
		fVal := decimal.NewFromInt(1)
		for _, r := range adjustedReserves {
			fVal = fVal.Mul(r.Sub(d))
		}
		fVal = fVal.Sub(k)

		// f'(d) = -sum(product(adjustedReserves[j] - d for j != i))
		fPrime := decimal.Zero
		for i := range adjustedReserves {
			term := decimal.NewFromInt(1)
			for j, r := range adjustedReserves {
				if j != i {
					term = term.Mul(r.Sub(d))
				}
			}
			fPrime = fPrime.Add(term)
		}
		fPrime = fPrime.Neg()

		if fPrime.IsZero() {
			return decimal.Zero, nil, errors.New("Newton's method: zero derivative")
		}

		dNew := d.Sub(fVal.Div(fPrime))

		// Clamp to valid range
		if dNew.LessThan(decimal.Zero) {
			dNew = decimal.Zero
		}
		if dNew.GreaterThanOrEqual(minReserve) {
			dNew = minReserve.Sub(decimal.NewFromFloat(0.000001))
		}

		if dNew.Sub(d).Abs().LessThan(decimal.NewFromFloat(0.000001)) {
			d = dNew
			break
		}
		d = dNew
	}

	if d.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil, errors.New("calculated USDC out is non-positive")
	}

	newReserves = make([]decimal.Decimal, n)
	for i, r := range adjustedReserves {
		newReserves[i] = r.Sub(d)
	}

	return d, newReserves, nil
}

// GetPrices computes the price of each outcome.
// Price_i = product(all reserves except i) / sum(product(all reserves except j) for each j)
func GetPrices(pool *models.LiquidityPool) ([]decimal.Decimal, error) {
	reserves, err := ParseReserves(pool)
	if err != nil {
		return nil, err
	}

	n := len(reserves)
	if n < 2 {
		return nil, errors.New("need at least 2 outcomes")
	}

	// Compute product excluding each index
	prodExcluding := make([]decimal.Decimal, n)
	for i := 0; i < n; i++ {
		p := decimal.NewFromInt(1)
		for j := 0; j < n; j++ {
			if j != i {
				p = p.Mul(reserves[j])
			}
		}
		prodExcluding[i] = p
	}

	// Sum of all products
	total := decimal.Zero
	for _, p := range prodExcluding {
		total = total.Add(p)
	}

	if total.IsZero() {
		return nil, errors.New("total product sum is zero")
	}

	prices := make([]decimal.Decimal, n)
	for i := 0; i < n; i++ {
		prices[i] = prodExcluding[i].Div(total)
	}

	return prices, nil
}

// decimalSqrt computes the square root of a decimal using Newton's method.
func decimalSqrt(x decimal.Decimal) decimal.Decimal {
	if x.IsZero() {
		return decimal.Zero
	}
	if x.LessThan(decimal.Zero) {
		return decimal.Zero
	}

	// Initial guess
	guess := x.Div(decimal.NewFromInt(2))
	if guess.IsZero() {
		guess = decimal.NewFromFloat(0.1)
	}

	two := decimal.NewFromInt(2)
	epsilon := decimal.NewFromFloat(0.00000001)

	for i := 0; i < 100; i++ {
		// Newton: guess = (guess + x/guess) / 2
		newGuess := guess.Add(x.Div(guess)).Div(two)
		if newGuess.Sub(guess).Abs().LessThan(epsilon) {
			return newGuess
		}
		guess = newGuess
	}
	return guess
}
