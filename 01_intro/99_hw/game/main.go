package main

import (
	"fmt"
	"strings"
)

type Item struct {
	Name           string
	Place          string
	Wearable       bool
	GivesInventory bool
}

type Door struct {
	Open    bool
	KeyName string
}

type Room struct {
	Name          string
	Ways          []string
	Neighbors     map[string]*Room
	Items         []Item
	Interactables map[string]func(p *Player, itemName string) string
	EnterText     func(p *Player) string
	LookText      func(p *Player) string
	ExitCheck     map[string]func(p *Player) (ok bool, failMsg string)
}

func (r *Room) exitsLine() string {
	return "можно пройти - " + strings.Join(r.Ways, ", ")
}

func (r *Room) defaultLook() string {
	byPlace := make(map[string][]string)
	var placeOrder []string
	seen := make(map[string]bool)
	for _, it := range r.Items {
		if !seen[it.Place] {
			seen[it.Place] = true
			placeOrder = append(placeOrder, it.Place)
		}
		byPlace[it.Place] = append(byPlace[it.Place], it.Name)
	}
	if len(placeOrder) == 0 {
		return "пустая комната"
	}
	parts := make([]string, 0, len(placeOrder))
	for _, place := range placeOrder {
		parts = append(parts, "на "+place+": "+strings.Join(byPlace[place], ", "))
	}
	return strings.Join(parts, ", ")
}

func (r *Room) findItem(name string) (Item, int, bool) {
	for i, it := range r.Items {
		if it.Name == name {
			return it, i, true
		}
	}
	return Item{}, -1, false
}

func (r *Room) removeItemAt(idx int) {
	r.Items = append(r.Items[:idx], r.Items[idx+1:]...)
}

type Player struct {
	Room        *Room
	Inventory   map[string]bool
	HasBackpack bool
}

func (p *Player) hasItem(name string) bool {
	return p.Inventory[name]
}

func (p *Player) addItem(name string) {
	p.Inventory[name] = true
}

var (
	player   *Player
	commands map[string]func(p *Player, args []string) string
)

func main() {
	initGame()
	_ = handleCommand("осмотреться")
}

func initGame() {
	streetDoor := &Door{Open: false, KeyName: "ключи"}

	kitchen := &Room{
		Name:      "кухня",
		Ways:      []string{"коридор"},
		Neighbors: map[string]*Room{},
		Items: []Item{
			{Name: "чай", Place: "столе"},
		},
		Interactables: map[string]func(*Player, string) string{},
		EnterText: func(_ *Player) string {
			return "кухня, ничего интересного"
		},
	}
	kitchen.LookText = func(p *Player) string {
		task := "надо собрать рюкзак и идти в универ"
		if p.HasBackpack {
			task = "надо идти в универ"
		}
		text := "ты находишься на кухне"
		if len(kitchen.Items) > 0 {
			text += ", " + kitchen.defaultLook()
		}
		return text + ", " + task
	}

	nothingInteresting := func(_ *Player) string {
		return "ничего интересного"
	}
	corridor := &Room{
		Name:      "коридор",
		Ways:      []string{"кухня", "комната", "улица"},
		Neighbors: map[string]*Room{},
		Interactables: map[string]func(*Player, string) string{
			"дверь": func(_ *Player, itemName string) string {
				if itemName != streetDoor.KeyName {
					return "не к чему применить"
				}
				streetDoor.Open = true
				return "дверь открыта"
			},
		},
		EnterText: nothingInteresting,
		LookText:  nothingInteresting,
		ExitCheck: map[string]func(*Player) (bool, string){
			"улица": func(_ *Player) (bool, string) {
				if !streetDoor.Open {
					return false, "дверь закрыта"
				}
				return true, ""
			},
		},
	}

	bedroom := &Room{
		Name:      "комната",
		Ways:      []string{"коридор"},
		Neighbors: map[string]*Room{},
		Items: []Item{
			{Name: "ключи", Place: "столе"},
			{Name: "конспекты", Place: "столе"},
			{Name: "рюкзак", Place: "стуле", Wearable: true, GivesInventory: true},
		},
		Interactables: map[string]func(*Player, string) string{},
		EnterText: func(_ *Player) string {
			return "ты в своей комнате"
		},
	}

	springOutside := func(_ *Player) string {
		return "на улице весна"
	}
	street := &Room{
		Name:          "улица",
		Ways:          []string{"домой"},
		Neighbors:     map[string]*Room{},
		Interactables: map[string]func(*Player, string) string{},
		EnterText:     springOutside,
		LookText:      springOutside,
	}

	kitchen.Neighbors["коридор"] = corridor
	corridor.Neighbors["кухня"] = kitchen
	corridor.Neighbors["комната"] = bedroom
	corridor.Neighbors["улица"] = street
	bedroom.Neighbors["коридор"] = corridor
	street.Neighbors["домой"] = corridor

	player = &Player{
		Room:        kitchen,
		Inventory:   map[string]bool{},
		HasBackpack: false,
	}

	commands = map[string]func(*Player, []string) string{
		"осмотреться": cmdLook,
		"идти":        cmdGo,
		"надеть":      cmdWear,
		"взять":       cmdTake,
		"применить":   cmdApply,
	}
}

func handleCommand(command string) string {
	parts := strings.Split(command, " ")
	if len(parts) == 0 || parts[0] == "" {
		return "неизвестная команда"
	}
	fn, ok := commands[parts[0]]
	if !ok {
		return "неизвестная команда"
	}
	return fn(player, parts[1:])
}

func cmdLook(p *Player, _ []string) string {
	r := p.Room
	var body string
	switch {
	case r.LookText != nil:
		body = r.LookText(p)
	default:
		body = r.defaultLook()
	}
	return body + ". " + r.exitsLine()
}

func cmdGo(p *Player, args []string) string {
	if len(args) < 1 {
		return "некуда идти"
	}
	dest := args[0]
	next, ok := p.Room.Neighbors[dest]
	if !ok {
		return fmt.Sprintf("нет пути в %s", dest)
	}
	if p.Room.ExitCheck != nil {
		if check, exists := p.Room.ExitCheck[dest]; exists {
			if okGo, msg := check(p); !okGo {
				return msg
			}
		}
	}
	p.Room = next
	enter := "ничего интересного"
	if next.EnterText != nil {
		enter = next.EnterText(p)
	}
	return enter + ". " + next.exitsLine()
}

func cmdWear(p *Player, args []string) string {
	if len(args) < 1 {
		return "нечего надевать"
	}
	it, idx, ok := p.Room.findItem(args[0])
	if !ok {
		return "нет такого"
	}
	if !it.Wearable {
		return "нельзя надеть"
	}
	p.Room.removeItemAt(idx)
	if it.GivesInventory {
		p.HasBackpack = true
	}
	return "вы надели: " + it.Name
}

func cmdTake(p *Player, args []string) string {
	if len(args) < 1 {
		return "нечего брать"
	}
	it, idx, ok := p.Room.findItem(args[0])
	if !ok {
		return "нет такого"
	}
	if it.Wearable {
		return "нельзя взять"
	}
	if !p.HasBackpack {
		return "некуда класть"
	}
	p.Room.removeItemAt(idx)
	p.addItem(it.Name)
	return "предмет добавлен в инвентарь: " + it.Name
}

func cmdApply(p *Player, args []string) string {
	if len(args) < 2 {
		return "не к чему применить"
	}
	itemName, target := args[0], args[1]
	if !p.hasItem(itemName) {
		return "нет предмета в инвентаре - " + itemName
	}
	fn, ok := p.Room.Interactables[target]
	if !ok || fn == nil {
		return "не к чему применить"
	}
	return fn(p, itemName)
}
