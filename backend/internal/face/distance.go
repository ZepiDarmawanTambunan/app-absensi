// Package face provides pure face-embedding math used by the attendance
// system: cosine distance, template averaging, and binary serialization.
//
// Architecture is on-device: the mobile app detects faces and extracts
// embeddings, and the server only stores templates and compares them.
// Raw face photos never reach the server.
package face

import (
	"encoding/binary"
	"errors"
	"math"
)

// SupportedDimensions lists the embedding sizes accepted by the API:
// 128 (face-api.js / dlib style), 192 (MobileFaceNet style),
// 512 (FaceNet / ArcFace style).
var SupportedDimensions = []int{128, 192, 512}

// IsSupportedDimension reports whether dim is an accepted embedding size.
func IsSupportedDimension(dim int) bool {
	for _, d := range SupportedDimensions {
		if d == dim {
			return true
		}
	}
	return false
}

// CosineDistance returns 1 - cosine similarity between a and b.
// Identical vectors score 0, orthogonal vectors 1, opposite vectors 2.
func CosineDistance(a, b []float32) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, errors.New("face: empty embedding")
	}
	if len(a) != len(b) {
		return 0, errors.New("face: embedding dimension mismatch")
	}
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0, errors.New("face: zero embedding")
	}
	sim := dot / (math.Sqrt(na) * math.Sqrt(nb))
	// Clamp float noise at the boundaries.
	if sim > 1 {
		sim = 1
	} else if sim < -1 {
		sim = -1
	}
	return 1 - sim, nil
}

// Average returns the L2-normalized mean of the given embeddings.
// All embeddings must share one non-zero dimension.
func Average(embeddings [][]float32) ([]float32, error) {
	if len(embeddings) == 0 {
		return nil, errors.New("face: no embeddings to average")
	}
	dim := len(embeddings[0])
	if dim == 0 {
		return nil, errors.New("face: empty embedding")
	}
	mean := make([]float32, dim)
	for _, e := range embeddings {
		if len(e) != dim {
			return nil, errors.New("face: embedding dimension mismatch")
		}
		for i, v := range e {
			mean[i] += v
		}
	}
	n := float32(len(embeddings))
	for i := range mean {
		mean[i] /= n
	}
	return Normalize(mean), nil
}

// Normalize returns the L2-normalized copy of v.
func Normalize(v []float32) []float32 {
	out := make([]float32, len(v))
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return out
	}
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}

// EncodeEmbedding serializes an embedding to little-endian float32 bytes
// for BLOB storage.
func EncodeEmbedding(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

// DecodeEmbedding parses bytes produced by EncodeEmbedding.
func DecodeEmbedding(b []byte) ([]float32, error) {
	if len(b) == 0 || len(b)%4 != 0 {
		return nil, errors.New("face: corrupt embedding blob")
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v, nil
}
