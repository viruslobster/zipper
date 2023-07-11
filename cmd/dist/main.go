package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"

	"github.com/viruslobster/zipper"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func speed(p float64, score int) float64 {
	e := 1.0 / p
	return float64(score) / e
}

func main() {
	agent := zipper.NewZipperAgent(10)
	if *cpuprofile != "" {
		f, err := os.Create("profile.cpu")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	pMap := make(map[int]float64)
	scores := make([]int, 0)
	for score := 50; score <= 2000; score += 50 {
		scores = append(scores, score)
		p := agent.PScore(6, score)
		pMap[score] = p

	}
	sort.Slice(scores, func(i, j int) bool {
		// return speed(pMap[scores[i]], scores[i]) > speed(pMap[scores[j]], scores[j])
		return scores[i] < scores[j]
	})
	for _, score := range scores {
		p := pMap[score]
		e := 1.0 / p
		fmt.Printf("%5d => %f, %f, %f\n", score, p, e, speed(p, score))
	}

}
