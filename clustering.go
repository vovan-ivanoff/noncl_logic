package main

import (
	"errors"
	"math"
)

type Point struct {
	X float64
	Y float64
}

type Cluster struct {
	Center Point
	Points []Point
}

func euclideanDistance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func arePointsEqual(a, b Point) bool {
	return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9
}

func KMeans(points []Point, k int) ([]Cluster, error) {
	if len(points) == 0 {
		return nil, errors.New("points are empty")
	}
	if k <= 0 {
		return nil, errors.New("k must be greater than 0")
	}
	if k > len(points) {
		return nil, errors.New("k cannot be greater than the number of points")
	}

	centers := make([]Point, k)
	for i := 0; i < k; i++ {
		idx := (len(points) * i) / k
		centers[i] = points[idx]
	}

	assignments := make([]int, len(points))
	for iter := 0; iter < 100; iter++ {
		changed := false
		for i, p := range points {
			bestIdx := 0
			bestDist := math.MaxFloat64
			for j, center := range centers {
				dist := euclideanDistance(p, center)
				if dist < bestDist {
					bestDist = dist
					bestIdx = j
				}
			}
			if assignments[i] != bestIdx {
				changed = true
			}
			assignments[i] = bestIdx
		}

		nextCenters := make([]Point, k)
		counts := make([]int, k)
		for i, p := range points {
			clusterIndex := assignments[i]
			nextCenters[clusterIndex].X += p.X
			nextCenters[clusterIndex].Y += p.Y
			counts[clusterIndex]++
		}

		for j := range nextCenters {
			if counts[j] == 0 {
				nextCenters[j] = centers[j]
				continue
			}
			nextCenters[j].X /= float64(counts[j])
			nextCenters[j].Y /= float64(counts[j])
		}

		if !changed {
			centers = nextCenters
			break
		}

		for j := range centers {
			if !arePointsEqual(centers[j], nextCenters[j]) {
				centers = nextCenters
				break
			}
		}
	}

	clusters := make([]Cluster, k)
	for i := range clusters {
		clusters[i].Center = centers[i]
	}

	for i, p := range points {
		idx := assignments[i]
		clusters[idx].Points = append(clusters[idx].Points, p)
	}

	return clusters, nil
}
