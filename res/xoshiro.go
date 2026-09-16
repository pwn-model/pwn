package res

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
)

// Xoshiro256pp is a Go port of Julia's default RNG, Xoshiro256++
// (JuliaLang/julia, stdlib/Random/src/Xoshiro.jl). It reproduces Julia's
// seeding (a SHA-256 hash of the seed, as in Random.hash_seed) and its
// generation step bit for bit, so that NewXoshiro256pp(seed).Uint64()
// matches rand(Random.Xoshiro(seed), UInt64) in Julia for the same seed.
//
// It implements math/rand/v2.Source, so it is a drop-in replacement for
// resource.Rand's original PCG source, and is used as such (see main.go)
// so that this implementation and the sibling Julia implementation, which
// uses its own default Xoshiro256++ RNG, produce matching sequences.
type Xoshiro256pp struct {
	s0, s1, s2, s3 uint64
}

// NewXoshiro256pp creates a new Xoshiro256pp seeded the way Julia's
// Xoshiro(seed) seeds itself, for a non-negative seed.
func NewXoshiro256pp(seed uint64) *Xoshiro256pp {
	digest := hashSeedJulia(seed)
	return &Xoshiro256pp{
		s0: binary.LittleEndian.Uint64(digest[0:8]),
		s1: binary.LittleEndian.Uint64(digest[8:16]),
		s2: binary.LittleEndian.Uint64(digest[16:24]),
		s3: binary.LittleEndian.Uint64(digest[24:32]),
	}
}

// hashSeedJulia reproduces Julia's Random.hash_seed(seed::Integer) for a
// non-negative seed that fits into a uint64.
func hashSeedJulia(seed uint64) [sha256.Size]byte {
	h := sha256.New()
	for {
		var word [4]byte
		binary.LittleEndian.PutUint32(word[:], uint32(seed))
		h.Write(word[:])
		seed >>= 32
		if seed == 0 {
			break
		}
	}
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

// Uint64 returns a uniformly-distributed random uint64 value, advancing
// the generator state the same way as Julia's xoshiro256++ step.
func (x *Xoshiro256pp) Uint64() uint64 {
	s0, s1, s2, s3 := x.s0, x.s1, x.s2, x.s3

	res := bits.RotateLeft64(s0+s3, 23) + s0

	t := s1 << 17
	s2 ^= s0
	s3 ^= s1
	s1 ^= s2
	s0 ^= s3
	s2 ^= t
	s3 = bits.RotateLeft64(s3, 45)

	x.s0, x.s1, x.s2, x.s3 = s0, s1, s2, s3
	return res
}
