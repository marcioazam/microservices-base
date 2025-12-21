package lens

import (
	"testing"

	"github.com/authcorp/libs/go/src/optics"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street string
	City   string
}

func PersonNameLens() optics.Lens[Person, string] {
	return optics.NewLens(
		func(p Person) string { return p.Name },
		func(p Person, name string) Person { p.Name = name; return p },
	)
}

func PersonAgeLens() optics.Lens[Person, int] {
	return optics.NewLens(
		func(p Person) int { return p.Age },
		func(p Person, age int) Person { p.Age = age; return p },
	)
}

func PersonAddressLens() optics.Lens[Person, Address] {
	return optics.NewLens(
		func(p Person) Address { return p.Address },
		func(p Person, addr Address) Person { p.Address = addr; return p },
	)
}

func AddressCityLens() optics.Lens[Address, string] {
	return optics.NewLens(
		func(a Address) string { return a.City },
		func(a Address, city string) Address { a.City = city; return a },
	)
}

// **Feature: resilience-lib-extraction, Property 20: Lens Get-Set Identity**
// **Validates: Requirements 65.5**
func TestLensGetSetIdentity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Get(Set(source, value)) == value", prop.ForAll(
		func(name string, age int, newName string) bool {
			lens := PersonNameLens()
			person := Person{Name: name, Age: age}
			updated := lens.Set(person, newName)
			return lens.Get(updated) == newName
		},
		gen.AnyString(),
		gen.Int(),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 21: Lens Set-Get Identity**
// **Validates: Requirements 65.5**
func TestLensSetGetIdentity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Set(source, Get(source)) == source", prop.ForAll(
		func(name string, age int) bool {
			lens := PersonNameLens()
			person := Person{Name: name, Age: age}
			updated := lens.Set(person, lens.Get(person))
			return updated.Name == person.Name && updated.Age == person.Age
		},
		gen.AnyString(),
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestLensBasicOperations(t *testing.T) {
	t.Run("Get retrieves value", func(t *testing.T) {
		lens := PersonNameLens()
		person := Person{Name: "Alice", Age: 30}
		if lens.Get(person) != "Alice" {
			t.Error("expected Alice")
		}
	})

	t.Run("Set creates new structure", func(t *testing.T) {
		lens := PersonNameLens()
		person := Person{Name: "Alice", Age: 30}
		updated := lens.Set(person, "Bob")
		if updated.Name != "Bob" {
			t.Error("expected Bob")
		}
		if person.Name != "Alice" {
			t.Error("original should be unchanged")
		}
	})

	t.Run("Modify applies function", func(t *testing.T) {
		lens := PersonAgeLens()
		person := Person{Name: "Alice", Age: 30}
		updated := lens.Modify(person, func(age int) int { return age + 1 })
		if updated.Age != 31 {
			t.Errorf("expected 31, got %d", updated.Age)
		}
	})
}

func TestLensComposition(t *testing.T) {
	t.Run("Compose creates nested lens", func(t *testing.T) {
		personAddr := PersonAddressLens()
		addrCity := AddressCityLens()
		personCity := optics.Compose(personAddr, addrCity)

		person := Person{
			Name:    "Alice",
			Address: Address{Street: "123 Main", City: "NYC"},
		}

		if personCity.Get(person) != "NYC" {
			t.Error("expected NYC")
		}

		updated := personCity.Set(person, "LA")
		if updated.Address.City != "LA" {
			t.Error("expected LA")
		}
	})
}

func TestIdentityLens(t *testing.T) {
	lens := optics.Identity[int]()
	if lens.Get(42) != 42 {
		t.Error("expected 42")
	}
	if lens.Set(42, 100) != 100 {
		t.Error("expected 100")
	}
}

func TestMapAtLens(t *testing.T) {
	lens := optics.MapAt("key", "default")
	m := map[string]string{"key": "value", "other": "data"}

	if lens.Get(m) != "value" {
		t.Error("expected value")
	}

	updated := lens.Set(m, "new")
	if updated["key"] != "new" {
		t.Error("expected new")
	}
	if m["key"] != "value" {
		t.Error("original should be unchanged")
	}

	// Test default
	empty := map[string]string{}
	if lens.Get(empty) != "default" {
		t.Error("expected default")
	}
}

func TestSliceAtLens(t *testing.T) {
	lens := optics.SliceAt(1, 0)
	s := []int{1, 2, 3}

	if lens.Get(s) != 2 {
		t.Error("expected 2")
	}

	updated := lens.Set(s, 42)
	if updated[1] != 42 {
		t.Error("expected 42")
	}
	if s[1] != 2 {
		t.Error("original should be unchanged")
	}

	// Test out of bounds
	if lens.Get([]int{}) != 0 {
		t.Error("expected default")
	}
}
