package main

import (
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"noncl_logic/chi2"
	"os"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/s-daehling/fyne-charts/pkg/coord"
	"github.com/s-daehling/fyne-charts/pkg/data"
)

type squareLayout struct{}

func (s *squareLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(200, 200)
	}
	return fyne.NewSize(400, 200)
}

func (s *squareLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) < 2 {
		return
	}
	plot := objects[0]
	controls := objects[1]
	squareSize := containerSize.Height
	if squareSize > containerSize.Width {
		squareSize = containerSize.Width
	}
	plot.Resize(fyne.NewSize(squareSize, squareSize))
	plot.Move(fyne.NewPos(0, 0))
	controlsWidth := containerSize.Width - squareSize
	if controlsWidth < 0 {
		controlsWidth = 0
	}
	controls.Resize(fyne.NewSize(controlsWidth, containerSize.Height))
	controls.Move(fyne.NewPos(squareSize, 0))
}

type curveInterval struct {
	A float64
	B float64
}

type distributionCurve struct {
	name   string
	color  fyne.ThemeColorName
	points []chi2.Vector2D
}

type markerBand struct {
	interval curveInterval
	color    color.NRGBA
}

func curveColor(name fyne.ThemeColorName) color.NRGBA {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x2c, G: 0x86, B: 0xff, A: 0x80}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0x80}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x22, G: 0xc5, B: 0x55, A: 0x80}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xf5, G: 0xb6, B: 0x00, A: 0x80}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x90, G: 0x90, B: 0x90, A: 0x80}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x91, G: 0x6b, B: 0xff, A: 0x80}
	default:
		return color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
	}
}

func globalRange(curves []distributionCurve) (float64, float64) {
	minX := math.MaxFloat64
	maxX := -math.MaxFloat64
	for _, curve := range curves {
		for _, point := range curve.points {
			if point.X < minX {
				minX = point.X
			}
			if point.X > maxX {
				maxX = point.X
			}
		}
	}
	if minX == math.MaxFloat64 || maxX == -math.MaxFloat64 {
		return 0, 1
	}
	return minX, maxX
}

func maxCurveHeight(points []chi2.Vector2D) float64 {
	maxY := 0.0
	for _, point := range points {
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	return maxY
}

func convertToPlotData(inp []chi2.Vector2D) []data.NumericalPoint {
	result := make([]data.NumericalPoint, 0, len(inp))
	for _, elem := range inp {
		result = append(result, data.NumericalPoint{N: elem.X, Val: elem.Y})
	}
	return result
}

func buildCurves() []distributionCurve {
	colors := []fyne.ThemeColorName{
		theme.ColorNamePrimary,
		theme.ColorNameError,
		theme.ColorNameSuccess,
		theme.ColorNameWarning,
		theme.ColorNameInputBackground,
		theme.ColorNameHover,
	}

	curves := make([]distributionCurve, 0, 50)
	for i := 1; i <= 50; i++ {
		series := chi2.GeneratSeries(i, 100)
		curves = append(curves, distributionCurve{
			name:   fmt.Sprintf("k=%d", i),
			color:  colors[(i-1)%len(colors)],
			points: series,
		})
	}
	return curves
}

func interpolateX(left, right chi2.Vector2D, threshold float64) float64 {
	if right.X == left.X {
		return left.X
	}
	if left.Y == right.Y {
		return left.X
	}
	ratio := (threshold - left.Y) / (right.Y - left.Y)
	return left.X + ratio*(right.X-left.X)
}

func visibleIntervals(points []chi2.Vector2D, alpha float64) ([]chi2.Vector2D, []curveInterval) {
	if len(points) == 0 {
		return nil, nil
	}

	ordered := append([]chi2.Vector2D(nil), points...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].X < ordered[j].X })

	maxY := 0.0
	for _, point := range ordered {
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	if maxY <= 0 {
		return nil, nil
	}

	threshold := alpha * maxY
	visible := make([]chi2.Vector2D, 0, len(ordered))
	for _, point := range ordered {
		if point.Y >= threshold {
			visible = append(visible, point)
		}
	}
	if len(visible) == 0 {
		return nil, nil
	}

	intervals := make([]curveInterval, 0)
	for i := 0; i < len(ordered); {
		if ordered[i].Y < threshold {
			i++
			continue
		}

		a := ordered[i].X
		if i > 0 && ordered[i-1].Y < threshold {
			a = interpolateX(ordered[i-1], ordered[i], threshold)
		}

		j := i
		for j+1 < len(ordered) && ordered[j+1].Y >= threshold {
			j++
		}

		b := ordered[j].X
		if j+1 < len(ordered) && ordered[j+1].Y < threshold {
			b = interpolateX(ordered[j], ordered[j+1], threshold)
		}

		intervals = append(intervals, curveInterval{A: a, B: b})
		i = j + 1
	}

	return visible, intervals
}

func filterIntervalsByLength(intervals []curveInterval, ratio float64) []curveInterval {
	if len(intervals) == 0 {
		return nil
	}

	ordered := append([]curveInterval(nil), intervals...)
	sort.Slice(ordered, func(i, j int) bool {
		leftLen := ordered[i].B - ordered[i].A
		rightLen := ordered[j].B - ordered[j].A
		if math.Abs(leftLen-rightLen) < 1e-9 {
			return ordered[i].A < ordered[j].A
		}
		return leftLen > rightLen
	})

	if ratio <= 0 {
		return ordered
	}
	if ratio >= 1 {
		if len(ordered) == 1 {
			return ordered
		}
		return ordered[:1]
	}

	keepRatio := 1.0 - ratio
	keepCount := int(math.Ceil(float64(len(ordered)) * keepRatio))
	if keepCount < 1 {
		keepCount = 1
	}
	if keepCount > len(ordered) {
		keepCount = len(ordered)
	}
	return ordered[:keepCount]
}

func collectVisibleIntervalsRaw(curves []distributionCurve, alpha float64) []curveInterval {
	result := make([]curveInterval, 0)
	for _, curve := range curves {
		_, intervals := visibleIntervals(curve.points, alpha)
		result = append(result, intervals...)
	}
	return result
}

func keepPercentageLabel(ratio float64) string {
	if ratio <= 0 {
		return "100%"
	}
	if ratio >= 1 {
		return "единственный"
	}
	return fmt.Sprintf("%.0f%%", (1.0-ratio)*100.0)
}

func filteredIntervalsForCurve(curve distributionCurve, alpha float64, lengthRatio float64) []curveInterval {
	_, intervals := visibleIntervals(curve.points, alpha)
	return filterIntervalsByLength(intervals, lengthRatio)
}

func buildChart(curves []distributionCurve, alpha float64, lengthRatio float64, clusters []IntervalCluster) *coord.CartesianNumericalChart {
	chart := coord.NewCartesianNumericalChart("Распределения χ²")
	chart.HideLegend()
	chart.SetYRange(0.0, 1.0)

	clusterPalette := []fyne.ThemeColorName{
		theme.ColorNamePrimary,
		theme.ColorNameError,
		theme.ColorNameSuccess,
		theme.ColorNameWarning,
		theme.ColorNameInputBackground,
		theme.ColorNameHover,
	}

	clusterIndexByInterval := make(map[string]int)
	for i, cluster := range clusters {
		for _, segment := range cluster.Intervals {
			key := fmt.Sprintf("%.6f:%.6f", segment.A, segment.B)
			clusterIndexByInterval[key] = i
		}
	}

	allVisibleIntervals := collectVisibleIntervalsRaw(curves, alpha)
	filteredKeys := make(map[string]bool)
	for _, segment := range filterIntervalsByLength(allVisibleIntervals, lengthRatio) {
		filteredKeys[fmt.Sprintf("%.6f:%.6f", segment.A, segment.B)] = true
	}

	for _, curve := range curves {
		visible, intervals := visibleIntervals(curve.points, alpha)
		if len(visible) == 0 {
			continue
		}

		seriesData := convertToPlotData(visible)
		series, err := coord.NewNumericalPointSeries(curve.name, curve.color, seriesData)
		if err != nil {
			panic(err)
		}
		if err := chart.AddLineSeries(series, false); err != nil {
			panic(err)
		}

		fixedLevel := 0.12
		for i, segment := range intervals {
			key := fmt.Sprintf("%.6f:%.6f", segment.A, segment.B)
			if !filteredKeys[key] {
				continue
			}
			colorName := curve.color
			if clusterIndex, ok := clusterIndexByInterval[key]; ok {
				colorName = clusterPalette[clusterIndex%len(clusterPalette)]
			}
			segmentPoints := []data.NumericalPoint{
				{N: segment.A, Val: fixedLevel},
				{N: segment.B, Val: fixedLevel},
			}
			segmentSeries, err := coord.NewNumericalPointSeries(fmt.Sprintf("%s-interval-%d", curve.name, i), colorName, segmentPoints)
			if err != nil {
				panic(err)
			}
			segmentSeries.SetLineWidth(5.0)
			if err := chart.AddLineSeries(segmentSeries, false); err != nil {
				panic(err)
			}
		}
	}

	return chart
}

func collectVisibleIntervals(curves []distributionCurve, alpha float64, lengthRatio float64) []Interval {
	result := make([]Interval, 0)
	for _, segment := range filterIntervalsByLength(collectVisibleIntervalsRaw(curves, alpha), lengthRatio) {
		result = append(result, Interval{A: segment.A, B: segment.B})
	}
	return result
}

func logVisibleIntervals(label string, curves []distributionCurve, alpha float64, lengthRatio float64) {
	intervals := collectVisibleIntervals(curves, alpha, lengthRatio)
	log.Printf("[ui] %s alpha=%.3f b=%.3f (оставляем %s самых длинных) intervals=%d", label, alpha, lengthRatio, keepPercentageLabel(lengthRatio), len(intervals))
	for i, segment := range intervals {
		log.Printf("  [%d] A=%.3f B=%.3f len=%.3f", i+1, segment.A, segment.B, segment.B-segment.A)
	}
}

func pointForInterval(interval Interval) Point {
	if interval.B < interval.A {
		interval.A, interval.B = interval.B, interval.A
	}
	center := (interval.A + interval.B) / 2.0
	length := interval.B - interval.A
	return Point{X: center, Y: length}
}

func buildRawPointChart(title string, intervals []Interval) *coord.CartesianNumericalChart {
	chart := coord.NewCartesianNumericalChart(title)
	chart.HideLegend()
	// chart.SetYRange(0.0, 1.0)

	pointsData := make([]data.NumericalPoint, 0, len(intervals))
	for _, interval := range intervals {
		p := pointForInterval(interval)
		pointsData = append(pointsData, data.NumericalPoint{N: p.X, Val: p.Y})
	}
	if len(pointsData) == 0 {
		return chart
	}

	series, err := coord.NewNumericalPointSeries("interval-points", theme.ColorNamePrimary, pointsData)
	if err != nil {
		panic(err)
	}
	series.SetDotSize(8)
	if err := chart.AddScatterSeries(series); err != nil {
		panic(err)
	}
	return chart
}

func buildClusterPointChart(title string, clusters []IntervalCluster) *coord.CartesianNumericalChart {
	chart := coord.NewCartesianNumericalChart(title)
	chart.HideLegend()
	// chart.SetYRange(0.0, 1.0)

	palette := []fyne.ThemeColorName{
		theme.ColorNamePrimary,
		theme.ColorNameError,
		theme.ColorNameSuccess,
		theme.ColorNameWarning,
		theme.ColorNameInputBackground,
		theme.ColorNameHover,
	}

	for i, cluster := range clusters {
		if len(cluster.Intervals) == 0 {
			continue
		}
		pointsData := make([]data.NumericalPoint, 0, len(cluster.Intervals))
		for _, interval := range cluster.Intervals {
			p := pointForInterval(interval)
			pointsData = append(pointsData, data.NumericalPoint{N: p.X, Val: p.Y})
		}
		series, err := coord.NewNumericalPointSeries(fmt.Sprintf("cluster-%d", i+1), palette[i%len(palette)], pointsData)
		if err != nil {
			panic(err)
		}
		series.SetDotSize(8)
		if err := chart.AddScatterSeries(series); err != nil {
			panic(err)
		}
	}
	return chart
}

type clusterSettings struct {
	count        int
	lengthWeight float64
}

func collectClusters(curves []distributionCurve, alpha float64, lengthRatio float64, cfg clusterSettings) []IntervalCluster {
	intervals := collectVisibleIntervals(curves, alpha, lengthRatio)
	if len(intervals) == 0 {
		return nil
	}
	if cfg.count <= 0 {
		cfg.count = 1
	}
	if cfg.count > len(intervals) {
		cfg.count = len(intervals)
	}
	clusters, err := ClusterIntervalsWithConfig(intervals, IntervalClusterConfig{
		K:           cfg.count,
		CenterScale: 1.0,
		LengthScale: cfg.lengthWeight,
	})
	if err != nil {
		fmt.Printf("[cluster] error=%v\n", err)
		return nil
	}
	return clusters
}

func formatClusterSummary(curves []distributionCurve, alpha float64, lengthRatio float64, cfg clusterSettings) string {
	intervals := collectVisibleIntervals(curves, alpha, lengthRatio)
	log.Printf("[cluster] alpha=%.3f, b=%.2f, visible intervals=%d, k=%d, lenScale=%.2f", alpha, lengthRatio, len(intervals), cfg.count, cfg.lengthWeight)
	if len(intervals) == 0 {
		log.Println("[cluster] no intervals for current threshold")
		return "Кластеры: интервалов нет"
	}

	clusters := collectClusters(curves, alpha, lengthRatio, cfg)
	if clusters == nil {
		return "Ошибка кластеризации"
	}

	parts := make([]string, 0, len(clusters))
	for i, cluster := range clusters {
		if len(cluster.Intervals) == 0 {
			continue
		}
		log.Printf("[cluster] group %d: count=%d center=%.3f length=%.3f", i+1, len(cluster.Intervals), cluster.Center, cluster.Length)
		parts = append(parts, fmt.Sprintf("Кластер %d: %d интервал(а), центр %.3f, длина %.3f",
			i+1,
			len(cluster.Intervals),
			cluster.Center,
			cluster.Length,
		))
	}
	if len(parts) == 0 {
		log.Println("[cluster] cluster result is empty")
		return "Кластеры: пустой результат"
	}
	return strings.Join(parts, "\n")
}

func main() {
	logFile, err := os.OpenFile("intervals.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	curves := buildCurves()
	alpha := 0.707
	lengthRatio := 0.0
	clusterCfg := clusterSettings{count: 4, lengthWeight: 1.0}

	a := app.New()
	w := a.NewWindow("chi2 distributions")
	w.Resize(fyne.NewSize(1200, 760))

	currentClusters := []IntervalCluster(nil)
	plotHolder := container.NewStack()
	plotHolder.Add(buildChart(curves, alpha, lengthRatio, currentClusters))

	rawPointsWindow := a.NewWindow("Точки интервалов")
	rawPointsWindow.Resize(fyne.NewSize(800, 600))
	clusteredPointsWindow := a.NewWindow("Кластеры интервалов")
	clusteredPointsWindow.Resize(fyne.NewSize(800, 600))

	alphaLabel := widget.NewLabel(fmt.Sprintf("α = %.2f", alpha))
	slider := widget.NewSlider(0.0, 1.0)
	slider.Step = 0.01
	slider.SetValue(alpha)

	clusterCountLabel := widget.NewLabel(fmt.Sprintf("Количество кластеров: %d", clusterCfg.count))
	clusterCountSlider := widget.NewSlider(1, 8)
	clusterCountSlider.Step = 1
	clusterCountSlider.SetValue(float64(clusterCfg.count))

	clusterWeightLabel := widget.NewLabel(fmt.Sprintf("Важность длины: %.2f", clusterCfg.lengthWeight))
	clusterWeightSlider := widget.NewSlider(0.2, 3.0)
	clusterWeightSlider.Step = 0.1
	clusterWeightSlider.SetValue(clusterCfg.lengthWeight)

	clusterLabel := widget.NewLabel("Кластеры: пока не рассчитаны")
	clusterLabel.Wrapping = fyne.TextWrapWord
	statusLabel := widget.NewLabel("Режим: интервалы по цветам графиков")
	statusLabel.Wrapping = fyne.TextWrapWord

	intervalFilterLabel := widget.NewLabel(fmt.Sprintf("Фильтр длины b = %.2f (оставляем %s самых длинных)", lengthRatio, keepPercentageLabel(lengthRatio)))
	intervalFilterSlider := widget.NewSlider(0.0, 1.0)
	intervalFilterSlider.Step = 0.01
	intervalFilterSlider.SetValue(lengthRatio)

	refreshPlotOnly := func() {
		statusLabel.SetText("Режим: интервалы по цветам графиков")
		clusterLabel.SetText("Кластеры: пока не рассчитаны")
		plotHolder.Objects = nil
		plotHolder.Add(buildChart(curves, alpha, lengthRatio, nil))
		plotHolder.Refresh()
	}
	refreshClusters := func() {
		currentClusters = collectClusters(curves, alpha, lengthRatio, clusterCfg)
		plotHolder.Objects = nil
		plotHolder.Add(buildChart(curves, alpha, lengthRatio, currentClusters))
		plotHolder.Refresh()
		clusterLabel.SetText(formatClusterSummary(curves, alpha, lengthRatio, clusterCfg))
		statusLabel.SetText("Режим: интервалы перекрашены по кластерам")
	}
	clusterButton := widget.NewButton("Кластеризовать интервалы", func() {
		log.Printf("[ui] button clicked, alpha=%.3f", alpha)
		refreshClusters()
		intervalsForPoints := collectVisibleIntervals(curves, alpha, lengthRatio)
		rawPointsWindow.SetContent(container.NewMax(buildRawPointChart("Точки интервалов", intervalsForPoints)))
		rawPointsWindow.Show()
		if len(currentClusters) > 0 {
			clusteredPointsWindow.SetContent(container.NewMax(buildClusterPointChart("Результат кластеризации", currentClusters)))
			clusteredPointsWindow.Show()
		} else {
			clusteredPointsWindow.Close()
		}
		log.Printf("[ui] cluster summary: %s", clusterLabel.Text)
	})

	slider.OnChanged = func(value float64) {
		alpha = value
		log.Printf("[ui] slider changed to alpha=%.3f", alpha)
		alphaLabel.SetText(fmt.Sprintf("α = %.2f", value))
		logVisibleIntervals("alpha update", curves, alpha, lengthRatio)
		refreshPlotOnly()
	}
	intervalFilterSlider.OnChanged = func(value float64) {
		lengthRatio = value
		log.Printf("[ui] slider changed to b=%.3f (оставляем %s самых длинных)", lengthRatio, keepPercentageLabel(lengthRatio))
		intervalFilterLabel.SetText(fmt.Sprintf("Фильтр длины b = %.2f (оставляем %s самых длинных)", lengthRatio, keepPercentageLabel(lengthRatio)))
		logVisibleIntervals("b update", curves, alpha, lengthRatio)
		refreshPlotOnly()
	}
	clusterCountSlider.OnChanged = func(value float64) {
		clusterCfg.count = int(value + 0.5)
		clusterCountLabel.SetText(fmt.Sprintf("Количество кластеров: %d", clusterCfg.count))
		refreshPlotOnly()
	}
	clusterWeightSlider.OnChanged = func(value float64) {
		clusterCfg.lengthWeight = value
		clusterWeightLabel.SetText(fmt.Sprintf("Важность длины: %.2f", clusterCfg.lengthWeight))
		refreshPlotOnly()
	}

	controls := container.NewVBox(
		widget.NewLabel("Порог выделения по альфа-уровню"),
		alphaLabel,
		slider,
		widget.NewLabel("Фильтр длины интервалов (b)"),
		intervalFilterLabel,
		intervalFilterSlider,
		widget.NewLabel("Параметры кластеризации"),
		clusterCountLabel,
		clusterCountSlider,
		clusterWeightLabel,
		clusterWeightSlider,
		clusterButton,
		statusLabel,
		widget.NewLabel("Кластеры интервалов:"),
		clusterLabel,
	)
	controls.Resize(fyne.NewSize(320, 560))

	content := container.New(&squareLayout{}, plotHolder, controls)
	w.SetContent(content)
	w.ShowAndRun()
}
