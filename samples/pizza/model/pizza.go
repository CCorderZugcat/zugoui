package model

import "encoding/gob"

type Topping struct {
	Topping string `bind:"topping"`
	Show    bool   `bind:">hidden;isZero"`
}

type Pizza struct {
	Size     string      `bind:"size"`
	Toppings [3]*Topping `bind:"toppings"`
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
