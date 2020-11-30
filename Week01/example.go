package main
import (
	"fmt"
	"github.com/pkg/errors"
)

func Func1() error {
	return errors.New("Func1 calling error")
}

func Func2 () error {
	return Func1()
}

func main() {
	fmt.Printf("%+v", Func2())
}

