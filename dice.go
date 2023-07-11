package zipper

import "math/rand"

// value([5]) = 485.239155

func Roll(n int) Dice {
	var result [6]int
	for i := 0; i < n; i++ {
		r := rand.Intn(6)
		result[r] += 1
	}
	return result
}

type Dice [6]int

func NewDice(dice []int) Dice {
	var result [6]int
	for _, d := range dice {
		if d < 1 || d > 6 {
			panic("dice not in range 1-6")
		}
		result[d-1]++
	}

	return result
}

// Freq returns the frequency of roll i in the dice
func (d Dice) Freq(i int) int {
	return d[i-1]
}

func (d *Dice) SetFreq(i, freq int) {
	d[i-1] = freq
}

// Mask returns the elements of dice specified by the bit mask,
// e.g. NewDice([]int{1,2,3}).Mask(0b101) = NewDice([]int{1, 3})
func (d Dice) Mask(mask int) (result Dice) {
	i := 0
	for val, freq := range d {
		for j := 0; j < freq; j++ {
			if mask&(1<<i) > 0 {
				result[val]++
			}
			i++
		}
	}
	return result
}

func (d Dice) Add(new Dice) Dice {
	for k, v := range new {
		d[k] += v
	}
	return d
}

func (d Dice) Sub(diff Dice) Dice {
	for k, v := range diff {
		d[k] -= v
		if d[k] < 0 {
			panic("invalid subtraction")
		}
	}
	return d
}

func (d Dice) Len() (result int) {
	for _, v := range d {
		result += v
	}
	return result
}

func (d Dice) Distinct() (result int) {
	for _, v := range d {
		if v > 0 {
			result += 1
		}
	}
	return result
}

func (d Dice) List() []int {
	result := make([]int, 0, 6)
	for k, v := range d {
		for i := 0; i < v; i++ {
			result = append(result, k+1)
		}
	}
	return result
}
