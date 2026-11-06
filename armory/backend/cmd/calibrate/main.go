package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"sort"

	"armory/internal/database"
)

type sample struct {
	user int64
	name string
	vec  []float64
}

func distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	i := int(math.Ceil(p*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	return sorted[i]
}

func rateAbove(sorted []float64, limit float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	i := sort.SearchFloat64s(sorted, math.Nextafter(limit, math.Inf(1)))
	return float64(len(sorted)-i) / float64(len(sorted))
}

func rateAtOrBelow(sorted []float64, limit float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	i := sort.SearchFloat64s(sorted, math.Nextafter(limit, math.Inf(1)))
	return float64(i) / float64(len(sorted))
}

func report(label string, size int, samples []sample, current float64) {
	var same, diff []float64
	for i := 0; i < len(samples); i++ {
		for j := i + 1; j < len(samples); j++ {
			d := distance(samples[i].vec, samples[j].vec)
			if samples[i].user == samples[j].user {
				same = append(same, d)
			} else {
				diff = append(diff, d)
			}
		}
	}
	sort.Float64s(same)
	sort.Float64s(diff)

	people := map[int64]bool{}
	for _, s := range samples {
		people[s.user] = true
	}
	fmt.Printf("\n%s (%d-d): %d vectors, %d people, %d same-person pairs, %d different-person pairs\n",
		label, size, len(samples), len(people), len(same), len(diff))
	if len(same) == 0 || len(diff) == 0 {
		fmt.Println("  not enough data: need at least two people with two or more vectors each")
		return
	}
	fmt.Printf("  same person:   min %.3f  median %.3f  p95 %.3f  max %.3f\n", same[0], percentile(same, 0.5), percentile(same, 0.95), same[len(same)-1])
	fmt.Printf("  different:     min %.3f  p1 %.3f  median %.3f  max %.3f\n", diff[0], percentile(diff, 0.01), percentile(diff, 0.5), diff[len(diff)-1])
	fmt.Printf("  current threshold %.2f: false reject %.1f%% of same pairs, false accept %.1f%% of different pairs\n",
		current, 100*rateAbove(same, current), 100*rateAtOrBelow(diff, current))

	if same[len(same)-1] < diff[0] {
		fmt.Printf("  clean gap: a threshold between %.3f and %.3f separates every pair\n", same[len(same)-1], diff[0])
		fmt.Printf("  suggested threshold: %.3f (midpoint)\n", (same[len(same)-1]+diff[0])/2)
		return
	}
	safe := diff[0] - 0.02
	fmt.Printf("  distributions overlap; for zero false accepts use a threshold under %.3f (rejects %.1f%% of same pairs)\n",
		diff[0], 100*rateAbove(same, safe))
	fmt.Printf("  suggested threshold: %.3f\n", safe)
}

func main() {
	path := flag.String("db", "", "path to a copy of the armory database")
	flag.Parse()
	if *path == "" {
		log.Fatal("usage: calibrate -db <copy of armory.db>")
	}

	db, err := database.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	enrollments, err := database.NewStore(db).ListFaceEnrollments(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	bySize := map[int][]sample{}
	for _, e := range enrollments {
		var vec []float64
		if json.Unmarshal([]byte(e.FaceRef), &vec) != nil || len(vec) == 0 {
			continue
		}
		bySize[len(vec)] = append(bySize[len(vec)], sample{user: e.User.ID, name: e.User.Name, vec: vec})
	}
	if len(bySize) == 0 {
		log.Fatal("no enrolled faces found")
	}

	sizes := make([]int, 0, len(bySize))
	for size := range bySize {
		sizes = append(sizes, size)
	}
	sort.Ints(sizes)
	for _, size := range sizes {
		label, current := "native model", 0.8
		if size == 128 {
			label, current = "face-api model", 0.5
		}
		report(label, size, bySize[size], current)
	}
}
