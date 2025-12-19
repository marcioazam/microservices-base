package tuple

import (
	"testing"
)

func TestPair(t *testing.T) {
	t.Run("NewPair creates pair", func(t *testing.T) {
		p := NewPair(1, "hello")
		if p.First != 1 || p.Second != "hello" {
			t.Error("unexpected values")
		}
	})

	t.Run("Unpack returns values", func(t *testing.T) {
		p := NewPair(1, "hello")
		a, b := p.Unpack()
		if a != 1 || b != "hello" {
			t.Error("unexpected values")
		}
	})

	t.Run("Swap swaps values", func(t *testing.T) {
		p := NewPair(1, "hello")
		swapped := p.Swap()
		if swapped.First != "hello" || swapped.Second != 1 {
			t.Error("unexpected values")
		}
	})
}

func TestPairMap(t *testing.T) {
	t.Run("MapFirst maps first value", func(t *testing.T) {
		p := NewPair(10, "hello")
		mapped := MapFirst(p, func(x int) int { return x * 2 })
		if mapped.First != 20 || mapped.Second != "hello" {
			t.Error("unexpected values")
		}
	})

	t.Run("MapSecond maps second value", func(t *testing.T) {
		p := NewPair(10, "hello")
		mapped := MapSecond(p, func(s string) int { return len(s) })
		if mapped.First != 10 || mapped.Second != 5 {
			t.Error("unexpected values")
		}
	})

	t.Run("MapBoth maps both values", func(t *testing.T) {
		p := NewPair(10, "hello")
		mapped := MapBoth(p,
			func(x int) int { return x * 2 },
			func(s string) int { return len(s) },
		)
		if mapped.First != 20 || mapped.Second != 5 {
			t.Error("unexpected values")
		}
	})
}

func TestTriple(t *testing.T) {
	t.Run("NewTriple creates triple", func(t *testing.T) {
		tr := NewTriple(1, "hello", true)
		if tr.First != 1 || tr.Second != "hello" || tr.Third != true {
			t.Error("unexpected values")
		}
	})

	t.Run("Unpack returns values", func(t *testing.T) {
		tr := NewTriple(1, "hello", true)
		a, b, c := tr.Unpack()
		if a != 1 || b != "hello" || c != true {
			t.Error("unexpected values")
		}
	})

	t.Run("ToPair returns first two values", func(t *testing.T) {
		tr := NewTriple(1, "hello", true)
		p := tr.ToPair()
		if p.First != 1 || p.Second != "hello" {
			t.Error("unexpected values")
		}
	})
}

func TestQuad(t *testing.T) {
	t.Run("NewQuad creates quad", func(t *testing.T) {
		q := NewQuad(1, "hello", true, 3.14)
		if q.First != 1 || q.Second != "hello" || q.Third != true || q.Fourth != 3.14 {
			t.Error("unexpected values")
		}
	})

	t.Run("Unpack returns values", func(t *testing.T) {
		q := NewQuad(1, "hello", true, 3.14)
		a, b, c, d := q.Unpack()
		if a != 1 || b != "hello" || c != true || d != 3.14 {
			t.Error("unexpected values")
		}
	})
}

func TestZip(t *testing.T) {
	t.Run("Zip combines slices", func(t *testing.T) {
		as := []int{1, 2, 3}
		bs := []string{"a", "b", "c"}
		pairs := Zip(as, bs)

		if len(pairs) != 3 {
			t.Errorf("expected 3 pairs, got %d", len(pairs))
		}
		if pairs[0].First != 1 || pairs[0].Second != "a" {
			t.Error("unexpected first pair")
		}
	})

	t.Run("Zip handles different lengths", func(t *testing.T) {
		as := []int{1, 2, 3, 4, 5}
		bs := []string{"a", "b"}
		pairs := Zip(as, bs)

		if len(pairs) != 2 {
			t.Errorf("expected 2 pairs, got %d", len(pairs))
		}
	})

	t.Run("Zip handles empty slices", func(t *testing.T) {
		as := []int{}
		bs := []string{"a", "b"}
		pairs := Zip(as, bs)

		if len(pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(pairs))
		}
	})
}

func TestUnzip(t *testing.T) {
	pairs := []Pair[int, string]{
		{First: 1, Second: "a"},
		{First: 2, Second: "b"},
		{First: 3, Second: "c"},
	}

	as, bs := Unzip(pairs)

	if len(as) != 3 || len(bs) != 3 {
		t.Error("unexpected lengths")
	}
	if as[0] != 1 || as[1] != 2 || as[2] != 3 {
		t.Error("unexpected first slice")
	}
	if bs[0] != "a" || bs[1] != "b" || bs[2] != "c" {
		t.Error("unexpected second slice")
	}
}

func TestZipWith(t *testing.T) {
	as := []int{1, 2, 3}
	bs := []int{10, 20, 30}
	result := ZipWith(as, bs, func(a, b int) int { return a + b })

	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}
	if result[0] != 11 || result[1] != 22 || result[2] != 33 {
		t.Error("unexpected results")
	}
}

func TestZip3(t *testing.T) {
	as := []int{1, 2}
	bs := []string{"a", "b"}
	cs := []bool{true, false}
	triples := Zip3(as, bs, cs)

	if len(triples) != 2 {
		t.Errorf("expected 2 triples, got %d", len(triples))
	}
	if triples[0].First != 1 || triples[0].Second != "a" || triples[0].Third != true {
		t.Error("unexpected first triple")
	}
}

func TestUnzip3(t *testing.T) {
	triples := []Triple[int, string, bool]{
		{First: 1, Second: "a", Third: true},
		{First: 2, Second: "b", Third: false},
	}

	as, bs, cs := Unzip3(triples)

	if len(as) != 2 || len(bs) != 2 || len(cs) != 2 {
		t.Error("unexpected lengths")
	}
	if as[0] != 1 || bs[0] != "a" || cs[0] != true {
		t.Error("unexpected values")
	}
}

func TestEnumerate(t *testing.T) {
	items := []string{"a", "b", "c"}
	enumerated := Enumerate(items)

	if len(enumerated) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(enumerated))
	}
	if enumerated[0].First != 0 || enumerated[0].Second != "a" {
		t.Error("unexpected first pair")
	}
	if enumerated[2].First != 2 || enumerated[2].Second != "c" {
		t.Error("unexpected last pair")
	}
}
