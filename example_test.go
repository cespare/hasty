package hasty_test

import (
	"fmt"
	"log"
	"regexp"

	"github.com/cespare/hasty"
)

func ExampleParse() {
	var person struct {
		Name string
		Age  int
	}
	data := `{Alice Smith, 44}`
	re := regexp.MustCompile(`^{(?P<Name>[\w ]+),\s+(?P<Age>\d+)}$`)
	if err := hasty.Parse([]byte(data), &person, re); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (%d)\n", person.Name, person.Age)
	// Output: Alice Smith (44)
}
