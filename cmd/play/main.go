package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/viruslobster/zipper"
)

type PlayerCPU struct {
	agent *zipper.ZipperAgent
	match zipper.Match
	score float64
}

func NewPlayerCPU() *PlayerCPU {
	return &PlayerCPU{
		agent: zipper.NewZipperAgent(20 /*maxDepth*/),
	}
}

func (p *PlayerCPU) RunningTotal() int {
	return int(p.match.Score)
}

func rerollCount(diceCount, matchCount int) int {
	count := diceCount - matchCount
	if count == 0 {
		return 6
	}
	return count
}
func (p *PlayerCPU) Roll() zipper.Dice {
	n := rerollCount(6, p.match.Used.Len())
	return zipper.Roll(n)
}

// Rolled tells the cpu what dice it rolled and returns the match
// it chose and the amount to increase its score by.
func (p *PlayerCPU) Rolled(dice zipper.Dice) (match zipper.Match, scoreWon int) {
	if p.match.Used.Len() == 6 {
		p.match.Used = zipper.Dice{}
	}
	match, expected, reroll := p.agent.BestMatch2(dice, p.match.Score, 1000-p.score)
	fmt.Printf("expected=%f\n", expected)
	fmt.Println(match)
	if match.Score == 0 && expected == 0 {
		// zipper
		p.match = zipper.Match{}
		return match, 0
	}
	p.match = p.match.Add(match.Used, match.Score)
	if match.Score > 0 && !reroll {
		// end the turn
		scoreDelta := p.match.Score
		p.score += scoreDelta
		p.match = zipper.Match{}
		return match, int(scoreDelta)
	}
	return match, 0
}

// roll => rolls all dice for you, cpu decides what to do
// roll a,b,c => rerolls the dice specified
// rolled a,b,c,d,e,f,g => specifies a roll

func parseCmd(str string) []string {
	str = strings.Trim(str, " \t\n")
	return strings.Split(str, " ")
}

func parseDice(str string) (zipper.Dice, error) {
	pieces := strings.Split(str, ",")
	var nums []int
	for _, piece := range pieces {
		i, err := strconv.ParseInt(piece, 10, 64)
		if err != nil {
			return [6]int{}, err
		}
		nums = append(nums, int(i))
	}
	return zipper.NewDice(nums), nil
}

func runFullGame() (turns int) {
	score := 0
	cpu := NewPlayerCPU()
	for score < 10000 {
		roll := cpu.Roll()
		_, scoreDelta := cpu.Rolled(roll)
		score += scoreDelta
		if scoreDelta > 0 {
			turns++
		}
	}
	return turns
}

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)

	cpu := NewPlayerCPU()

	score := 0
	for {
		fmt.Printf("> ")
		raw, _ := reader.ReadString('\n')
		input := parseCmd(raw)
		switch input[0] {
		case "r", "roll":
			roll := cpu.Roll()
			fmt.Printf("Rolled %d dice: %v\n", roll.Len(), roll.List())
			match, scoreDelta := cpu.Rolled(roll)
			score += scoreDelta
			fmt.Printf("match: score=%d, used=%v\n", int(match.Score), match.Used.List())
			if match.Score == 0 {
				fmt.Println("zipper")
				fmt.Printf("score = %d\n", score)
			} else if scoreDelta > 0 {
				fmt.Printf("won   = %d\n", scoreDelta)
				fmt.Printf("score = %d\n", score)
				fmt.Println("end turn")
			} else {
				fmt.Printf("running total = %d\n", cpu.RunningTotal())
				fmt.Printf("Reroll %d dice\n", rerollCount(roll.Len(), match.Used.Len()))
			}

		case "exit", "quit":
			return
		case "go":
			fmt.Println(runFullGame())
		default:
			fmt.Println("Invalid command")
		}
	}
}
