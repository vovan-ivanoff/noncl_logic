package main

import "errors"

// Interval is a segment on the x-axis.
type Interval struct {
	A float64
	B float64
}

// IntervalCluster groups intervals that are close by center and length.
type IntervalCluster struct {
	Center    float64
	Length    float64
	Intervals []Interval
}

type IntervalClusterConfig struct {
	K           int
	CenterScale float64
	LengthScale float64
}

func ClusterIntervalsWithConfig(intervals []Interval, cfg IntervalClusterConfig) ([]IntervalCluster, error) {
	if len(intervals) == 0 {
		return nil, errors.New("intervals are empty")
	}
	if cfg.K <= 0 {
		return nil, errors.New("k must be greater than 0")
	}
	if cfg.K > len(intervals) {
		return nil, errors.New("k cannot be greater than the number of intervals")
	}
	if cfg.CenterScale <= 0 {
		cfg.CenterScale = 1.0
	}
	if cfg.LengthScale <= 0 {
		cfg.LengthScale = 1.0
	}

	points := make([]Point, 0, len(intervals))
	for _, interval := range intervals {
		if interval.B < interval.A {
			interval.A, interval.B = interval.B, interval.A
		}
		center := (interval.A + interval.B) / 2.0
		length := interval.B - interval.A
		points = append(points, Point{
			X: center * cfg.CenterScale,
			Y: length * cfg.LengthScale,
		})
	}

	clusters, err := KMeans(points, cfg.K)
	if err != nil {
		return nil, err
	}

	result := make([]IntervalCluster, 0, len(clusters))
	for _, cluster := range clusters {
		if len(cluster.Points) == 0 {
			continue
		}
		group := IntervalCluster{
			Center:    cluster.Center.X / cfg.CenterScale,
			Length:    cluster.Center.Y / cfg.LengthScale,
			Intervals: make([]Interval, 0, len(cluster.Points)),
		}
		for _, p := range cluster.Points {
			center := p.X / cfg.CenterScale
			length := p.Y / cfg.LengthScale
			start := center - length/2.0
			end := center + length/2.0
			group.Intervals = append(group.Intervals, Interval{A: start, B: end})
		}
		result = append(result, group)
	}

	return result, nil
}

// ClusterIntervals groups similar intervals by their midpoint and length.
// The feature vector is: (center, length), so intervals are clustered by both
// position and size, which is useful for alpha-selected segments.
func ClusterIntervals(intervals []Interval, k int) ([]IntervalCluster, error) {
	return ClusterIntervalsWithConfig(intervals, IntervalClusterConfig{K: k, CenterScale: 1.0, LengthScale: 1.0})
}
