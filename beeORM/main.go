package main

import (
	"fmt"
	"reflect"
)

type Account struct{}

func main() {
	typ := reflect.Indirect(reflect.ValueOf(&Account{})).Type()
	fmt.Println(typ.Name()) // Account
}
