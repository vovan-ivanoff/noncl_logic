package main

import "testing"

func TestKMeansClusters(t *testing.T) {
	points := []Point{
		{X: 0, Y: 0},
		{X: 0, Y: 1},
		{X: 10, Y: 10},
		{X: 10, Y: 11},
	}

	clusters, err := KMeans(points, 2)
	if err != nil {
		t.Fatalf("KMeans returned error: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
}

func TestClusterIntervals(t *testing.T) {
	intervals := []Interval{
		{A: 0, B: 2},
		{A: 0.2, B: 2.1},
		{A: 10, B: 12},
		{A: 9.8, B: 11.9},
	}

	clusters, err := ClusterIntervals(intervals, 2)
	if err != nil {
		t.Fatalf("ClusterIntervals returned error: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 interval clusters, got %d", len(clusters))
	}
	if len(clusters[0].Intervals) == 0 && len(clusters[1].Intervals) == 0 {
		t.Fatal("expected at least one interval in a cluster")
	}
}

func TestFilterIntervalsByLength(t *testing.T) {
	intervals := []curveInterval{
		{A: 0, B: 1},
		{A: 2, B: 5},
		{A: 6, B: 10},
	}

	if got := filterIntervalsByLength(intervals, 0); len(got) != len(intervals) {
		t.Fatalf("ratio 0 should keep the full list, got %d intervals", len(got))
	}

	if got := filterIntervalsByLength(intervals, 1); len(got) != 1 {
		t.Fatalf("ratio 1 should keep exactly one interval, got %d intervals", len(got))
	}

	if got := filterIntervalsByLength(intervals, 0.5); len(got) != 2 {
		t.Fatalf("ratio 0.5 should keep the two longest intervals, got %d intervals", len(got))
	}
}
