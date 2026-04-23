package model

import (
	"encoding/gob"

	_ "github.com/CCorderZugcat/zugoui/observable/controllers/scroll"
)

type Topping struct {
	Topping string `bind:"topping"`
	Show    bool   `bind:">hidden;isZero"`
}

type Pizza struct {
	Size     string     `bind:"size"`
	Toppings []*Topping `bind:"toppings" controller:"scroll,3"`
}

type Buttons struct {
	Up   bool `bind:"toppings.up>disabled"`
	Down bool `bind:"toppings.down>disabled"`
}

func init() {
	gob.Register(new(Topping))
	gob.Register(new(Pizza))
	gob.Register(new(Buttons))
}
