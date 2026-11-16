package chi2

import (
	"math"
	"math/rand/v2"
)

type Vector2D struct {
	X float64
	Y float64
}

// ChiSquaredBoxMuller генерирует одну величину хи-квадрат с k степенями свободы
func ChiSquaredBoxMuller(k int) float64 {
	sum := 0.0

	// Генерируем k стандартных нормальных величин и суммируем их квадраты
	for i := 0; i < k; i += 2 {
		// math/rand/v2.Float64() возвращает равномерно распределенное число в [0.0, 1.0)
		u1 := rand.Float64()
		u2 := rand.Float64()

		// Защита от логарифма нуля
		if u1 == 0 {
			u1 = 1e-15
		}

		// Преобразование Бокса — Мюллера дает сразу две нормальные величины
		r := math.Sqrt(-2.0 * math.Log(u1))
		theta := 2.0 * math.Pi * u2

		z1 := r * math.Cos(theta)
		sum += z1 * z1

		// Если k нечетное, используем только первую величину z1 на последней итерации
		if i+1 < k {
			z2 := r * math.Sin(theta)
			sum += z2 * z2
		}
	}

	return sum
}

func Density(x float64, k int) float64 {
	if x <= 0 {
		return 0
	}
	return math.Exp(-x/2.0) *
		math.Pow(x, float64(k)/2.0-1.0) /
		(math.Pow(2.0, float64(k)/2.0) * math.Gamma(float64(k)/2.0))
}

func GeneratSeries(k int, n int) []Vector2D {
	if n <= 1 {
		return nil
	}

	result := make([]Vector2D, 0, n)
	maxX := 20.0 + float64(k)*2.5
	step := maxX / float64(n-1)

	for i := 0; i < n; i++ {
		x := step * float64(i)
		result = append(result, Vector2D{X: x, Y: Density(x, k)})
	}
	return result
}
