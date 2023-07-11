package zipper

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

/* Zipper Score
 * 5 => 50
 * 1 => 100
 * Three of a kind => 100 * r
 * Three 1's => 1000
 * Three pairs => 750
 * Straight => 1500
 * 6 of a kind => 2000
 */

// score returns the score coresponding to a dice roll
// that is an *exact* match
func score(roll Dice) float64 {
	// roll.Normalize()
	var (
		count    = roll.Len()
		distinct = roll.Distinct()
	)
	if count == 6 && distinct == 1 {
		return 1500 // straight
	}
	if count == 6 && distinct == 6 {
		return 2000 // six of a kind
	}
	if count == 6 {
		pairs := 0
		for _, v := range roll {
			if v == 2 {
				pairs += 1
			}
		}
		if pairs == 3 {
			return 750 // three pairs
		}
	}
	if count == 3 && roll.Freq(1) == 3 {
		return 1000 // three ones
	}
	if count == 3 && distinct == 1 {
		for i := 1; i <= 6; i++ {
			if roll.Freq(i) > 0 {
				return float64(i) * 100 // three of a kind (not ones)
			}
		}
		panic("unreachable")
	}
	if count == 1 && roll.Freq(1) == 1 {
		return 100
	}
	if count == 1 && roll.Freq(5) == 1 {
		return 50
	}
	return 0
}

// Match represents a subset of dice used to score in a turn
type Match struct {
	Score float64
	Used  Dice
}

func (c Match) Add(roll Dice, points float64) Match {
	c.Score += points
	c.Used = c.Used.Add(roll)
	return c
}

func scoringCombos(dice Dice) []Dice {
	count := 1 << dice.Len()
	found := make(map[Dice]bool)
	for mask := 1; mask < count; mask++ {
		combo := dice.Mask(mask)
		found[combo] = found[combo] || score(combo) > 0
	}
	result := make([]Dice, 0, len(found))
	for combo, ok := range found {
		if !ok {
			continue
		}
		result = append(result, combo)
	}
	return result
}

func getMatches(roll Dice) []Match {
	raw := getMatchesImpl(roll, Match{})
	matchByDice := make(map[Dice]Match)
	// first result is always zipper
	for _, newMatch := range raw[1:] {
		match := matchByDice[newMatch.Used]
		if newMatch.Score > match.Score {
			matchByDice[newMatch.Used] = newMatch
		}
	}
	result := make([]Match, 0, len(matchByDice))
	for _, match := range matchByDice {
		result = append(result, match)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})
	return result
}

func getMatchesImpl(roll Dice, match Match) []Match {
	var result []Match
	result = append(result, match)
	if roll.Len() == 0 {
		return result
	}
	for _, combo := range scoringCombos(roll) {
		newMatch := match.Add(combo, score(combo))
		unused := roll.Sub(combo)
		result = append(result, getMatchesImpl(unused, newMatch)...)
	}
	return result
}

type ZipperAgent struct {
	possibleRollsCache map[int][]Dice
	expectedCache      map[[2]int][2]float64
	pScoreCache        map[[2]int]float64
	expectedTurnsCache map[[3]int]float64

	maxDepth int
}

func NewZipperAgent(maxDepth int) *ZipperAgent {
	return &ZipperAgent{
		possibleRollsCache: make(map[int][]Dice),
		expectedCache:      make(map[[2]int][2]float64),
		pScoreCache:        make(map[[2]int]float64),
		expectedTurnsCache: make(map[[3]int]float64),
		maxDepth:           maxDepth,
	}
}

// BestMatch returns the best match to take given the dice rolled
// and the score already on the line. The expected value of choosing
// that match is also returned.
func (z *ZipperAgent) BestMatch(dice Dice, delta float64) (Match, float64) {
	return z.bestMatchImpl(dice, delta, 0)
}

// expected returns the expected value of rolling numDice dice.
// It also returns the probability you score after rolling numDice dice.
func (z *ZipperAgent) expected(numDice int, depth int) (expected, p float64) {
	// return if we already computed expected(numDice) at depth or higher
	for d := 0; d <= depth; d++ {
		result, ok := z.expectedCache[[2]int{numDice, d}]
		if ok {
			return result[0], result[1]
		}
	}
	expected, p = z.expectedImpl(numDice, depth)
	z.expectedCache[[2]int{numDice, depth}] = [2]float64{expected, p}
	return expected, p
}

func rerollCount(diceCount, matchCount int) int {
	count := diceCount - matchCount
	if count == 0 {
		return 6
	}
	return count
}

// TODO: hack
func GetMatches(dice Dice) []Match {
	return getMatches(dice)
}

func (z *ZipperAgent) BestMatch2(dice Dice, pot, goal float64) (Match, float64, bool) {
	matches := getMatches(dice)
	if len(matches) == 0 || pot+matches[0].Score > goal {
		return Match{}, 0, false
	}

	best := matches[0]
	bestValue := z.expectedTurns(6, 0, goal-pot-matches[0].Score) + 1
	reroll := false
	if pot+matches[0].Score == goal {
		return matches[0], 0.0, false
	}
	for _, match := range getMatches(dice) {
		c := rerollCount(dice.Len(), match.Used.Len())
		val := z.expectedTurns(c, pot+match.Score, goal)
		if val < bestValue {
			best = match
			bestValue = val
			reroll = true
		}
	}
	return best, bestValue, reroll
}

func (z *ZipperAgent) expectedTurns(numDice int, pot, goal float64) (result float64) {
	key := [3]int{numDice, int(pot), int(goal)}
	result, ok := z.expectedTurnsCache[key]
	if !ok {
		result = z.expectedTurnsImpl(numDice, pot, goal)
		z.expectedTurnsCache[key] = result
	}
	return result
}

func (z *ZipperAgent) expectedTurnsImpl(numDice int, pot, goal float64) (result float64) {
	for _, roll := range z.possibleRolls(numDice) {
		match, val, _ := z.BestMatch2(roll, pot, goal)
		if match.Score > 0 {
			result += val * pDice(roll)
		} else {
			result += pDice(roll) * (goal / 1000)
		}
	}
	return result
}

func (z *ZipperAgent) bestMatchImpl(dice Dice, delta float64, depth int) (Match, float64) {
	var best Match
	var bestValue float64
	for i, match := range getMatches(dice) {
		c := rerollCount(dice.Len(), match.Used.Len())
		exp, p := z.expected(c, depth)
		value := p*(match.Score+delta) + exp
		// the first match uses all scoring dice. You can choose to not reroll.
		if i == 0 {
			value = math.Max(match.Score+delta, value)
		}
		if depth == 0 {
			fmt.Printf("value(%v) = %f\n", match.Used.List(), value)
		}
		if value > bestValue {
			best = match
			bestValue = value
		}
	}
	return best, bestValue
}

// TODO: can't return p here, its affected by recursion
func (z *ZipperAgent) expectedImpl(numDice int, depth int) (expected, p float64) {
	if depth >= z.maxDepth {
		return 0, 0
	}
	rolls := z.possibleRolls(numDice)
	pScore := 0.0
	for _, roll := range rolls {
		_, value := z.bestMatchImpl(roll, 0, depth+1)
		if value > 0 {
			p := pDice(roll)
			expected += value * p
			pScore += p
		}
	}
	return expected, pScore
}

// possibleRolls calculates all possible combinations of rolling
// numDice dice
func (z *ZipperAgent) possibleRolls(numDice int) []Dice {
	rolls, ok := z.possibleRollsCache[numDice]
	if !ok {
		var err error
		rolls, err = possibleRollsImpl(1, 0, numDice)
		if err != nil {
			panic(err)
		}
		z.possibleRollsCache[numDice] = rolls
	}
	return rolls
}

func possibleRollsImpl(dieVal, len, goalLen int) (result []Dice, err error) {
	if dieVal > 6 {
		if len != goalLen {
			return nil, errors.New("unsatisfiable")
		}
		return []Dice{Dice{}}, nil
	}
	for i := 0; i <= goalLen-len; i++ {
		var dice Dice
		dice.SetFreq(dieVal, i)
		subDice, err := possibleRollsImpl(dieVal+1, len+i, goalLen)
		if err != nil {
			continue
		}
		for _, sub := range subDice {
			result = append(result, dice.Add(sub))
		}
	}
	return result, nil
}

// Distribution returns the probability of reaching
// different scores in a single turn
// numDice  - The number of dice you start rolling with
// score    - The score you have already won for the turn
// maxScore - The highest score to compute a probability for.
//			  Probabilities are computed for scores [0, maxScore]
//			  in increments of 50.
func (z *ZipperAgent) PScore(numDice int, score int) float64 {
	key := [2]int{numDice, score}
	result, ok := z.pScoreCache[key]
	if !ok {
		result = z.pScoreImpl(numDice, score)
		z.pScoreCache[key] = result
	}
	return result
}

func (z *ZipperAgent) pScoreImpl(numDice int, score int) (result float64) {
	if score <= 0 {
		return 0
	}
	for _, roll := range z.possibleRolls(numDice) {
		var bestP float64
		prob := pDice(roll)
		for i, match := range getMatches(roll) {
			if int(match.Score) > score {
				continue
			}
			// you can take the first match and stop rolling
			if i == 0 && int(match.Score) == score {
				bestP = prob
				break
			}

			// for the other matches you must keep rolling
			c := rerollCount(numDice, match.Used.Len())
			p := prob * z.PScore(c, score-int(match.Score))
			if p > bestP {
				bestP = p
			}
		}
		result += bestP
	}
	return result
}

func (z *ZipperAgent) Dist1Turn(numDice int) []float64 {
	dist := make(map[int]float64)
	for _, roll := range z.possibleRolls(numDice) {

	}

}

func (z *ZipperAgent) WTF() {
	z.maxDepth = 2
	z.expectedCache = make(map[[2]int][2]float64)
	numDice := 6
	rolls := z.possibleRolls(numDice)
	pScore := 0.0
	nScore := 0
	for _, roll := range rolls {
		matches := getMatches(roll)
		if len(matches) > 0 {
			p := pDice(roll)
			fmt.Printf("p(%v), %d = %f\n", roll.List(), roll.Distinct(), p)
			pScore += p
			nScore++
		}
	}

	N := 100000
	hits := 0
	scoreSet := make(map[Dice]bool)
	for n := 0; n < N; n++ {
		dice := Roll(numDice)
		if len(getMatches(dice)) > 0 {
			hits++
			scoreSet[dice] = true
		}
	}
	fmt.Printf("%f, %d\n", pScore, nScore)
	fmt.Printf("%f, %d\n", float64(hits)/float64(N), len(scoreSet))
}

func factorial(x int) float64 {
	if x < 0 {
		panic("factorial only defined over positive integers")
	} else if x == 0 {
		return 1
	}
	result := 1.0
	x_float := float64(x)
	for x_float > 0 {
		result *= x_float
		x_float -= 1
	}
	return result
}

// pDice returns the probability of rolling dice
func pDice(dice Dice) float64 {
	num := factorial(dice.Len())
	for _, freq := range dice {
		num /= factorial(freq)
	}
	denom := math.Pow(6, float64(dice.Len()))
	return num / denom
}

// 1.158009
