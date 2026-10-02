package face

import (
	"math"
	"testing"
)

func approx(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCosineDistance(t *testing.T) {
	t.Run("identical vectors score 0", func(t *testing.T) {
		a := []float32{0.2, -0.5, 0.8, 0.1}
		d, err := CosineDistance(a, a)
		if err != nil {
			t.Fatal(err)
		}
		approx(t, d, 0)
	})

	t.Run("orthogonal vectors score 1", func(t *testing.T) {
		a := []float32{1, 0, 0, 0}
		b := []float32{0, 1, 0, 0}
		d, err := CosineDistance(a, b)
		if err != nil {
			t.Fatal(err)
		}
		approx(t, d, 1)
	})

	t.Run("opposite vectors score 2", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{-1, -2, -3}
		d, err := CosineDistance(a, b)
		if err != nil {
			t.Fatal(err)
		}
		approx(t, d, 2)
	})

	t.Run("scale invariant", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{10, 20, 30}
		d, err := CosineDistance(a, b)
		if err != nil {
			t.Fatal(err)
		}
		approx(t, d, 0)
	})

	t.Run("dimension mismatch errors", func(t *testing.T) {
		if _, err := CosineDistance([]float32{1, 2}, []float32{1, 2, 3}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty embedding errors", func(t *testing.T) {
		if _, err := CosineDistance(nil, []float32{1}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("zero embedding errors", func(t *testing.T) {
		if _, err := CosineDistance([]float32{0, 0}, []float32{1, 1}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAverage(t *testing.T) {
	t.Run("mean of identical vectors is itself normalized", func(t *testing.T) {
		v := []float32{3, 4}
		avg, err := Average([][]float32{v, v, v})
		if err != nil {
			t.Fatal(err)
		}
		approx(t, float64(avg[0]), 0.6)
		approx(t, float64(avg[1]), 0.8)
	})

	t.Run("result is L2-normalized", func(t *testing.T) {
		avg, err := Average([][]float32{{1, 0}, {0, 1}})
		if err != nil {
			t.Fatal(err)
		}
		var sum float64
		for _, x := range avg {
			sum += float64(x) * float64(x)
		}
		approx(t, math.Sqrt(sum), 1)
	})

	t.Run("dimension mismatch errors", func(t *testing.T) {
		if _, err := Average([][]float32{{1, 2}, {1, 2, 3}}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty input errors", func(t *testing.T) {
		if _, err := Average(nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEmbeddingCodec(t *testing.T) {
	t.Run("encode/decode roundtrip", func(t *testing.T) {
		v := make([]float32, 128)
		for i := range v {
			v[i] = float32(i)*0.01 - 0.5
		}
		b := EncodeEmbedding(v)
		if len(b) != 4*128 {
			t.Fatalf("expected 512 bytes, got %d", len(b))
		}
		back, err := DecodeEmbedding(b)
		if err != nil {
			t.Fatal(err)
		}
		if len(back) != len(v) {
			t.Fatalf("expected %d floats, got %d", len(v), len(back))
		}
		for i := range v {
			if back[i] != v[i] {
				t.Fatalf("mismatch at %d", i)
			}
		}
	})

	t.Run("corrupt blob errors", func(t *testing.T) {
		if _, err := DecodeEmbedding([]byte{1, 2, 3}); err == nil {
			t.Fatal("expected error")
		}
		if _, err := DecodeEmbedding(nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSupportedDimensions(t *testing.T) {
	for _, d := range []int{128, 192, 512} {
		if !IsSupportedDimension(d) {
			t.Fatalf("expected %d to be supported", d)
		}
	}
	if IsSupportedDimension(64) {
		t.Fatal("64 should not be supported")
	}
}
