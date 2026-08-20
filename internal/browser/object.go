package browser

import (
	"syscall/js"
)

type Object struct {
	value js.Value
}

func Global() *Object {
	return &Object{
		value: js.Global(),
	}
}

func Document() *Object {
	return &Object{
		value: Global().value.Get("document"),
	}
}

func DocumentElement() *Object {
	return &Object{
		value: Global().value.Get("document").Get("documentElement"),
	}
}

const (
	EventClick = "click"
	EventInput = "input"
)

func Alert(text string) {
	Global().value.Call("alert", text)
}

func (o *Object) AddEventHandler(event string, h func(*Object, *Object) any) {
	o.value.Call("addEventListener", event, js.FuncOf(
		func(this js.Value, args []js.Value) any {
			return h(&Object{value: this}, &Object{value: args[0]})
		},
	))
}

func (o *Object) AddClickHandler(h func(*Object, *Object) any) {
	o.AddEventHandler(EventClick, h)
}

func (o *Object) AddInputHandler(h func(*Object, *Object) any) {
	o.AddEventHandler(EventInput, h)
}

func (o *Object) PreventDefault() {
	o.value.Call("preventDefault")
}

func (o *Object) StopPropagation() {
	o.value.Call("stopPropagation")
}

func (o *Object) GetElementByID(id string) *Object {
	return &Object{
		value: o.value.Call("getElementById", id),
	}
}

func (o *Object) Parent() *Object {
	parent := o.value.Get("parentElement")
	if parent.IsNull() {
		return nil
	}

	return &Object{
		value: parent,
	}
}

func (o *Object) GetAttribute(attr string) string {
	value := o.value.Call("getAttribute", attr)
	if value.Type() == js.TypeString {
		return value.String()
	}

	return ""
}

func (o *Object) SetAttribute(attr, value string) {
	o.value.Call("setAttribute", attr, value)
}

func (o *Object) InsertBefore(newObject, referenceObject *Object) {
	o.value.Call("insertBefore", newObject.value, referenceObject.value)
}

func (o *Object) HasClass(className string) bool {
	result := o.value.Get("classList").Call("contains", className)

	if result.Type() == js.TypeBoolean {
		return result.Bool()
	}

	return false
}

func (o *Object) AddClass(className string) {
	o.value.Get("classList").Call("add", className)
}

func (o *Object) ToggleClass(className string) {
	o.value.Get("classList").Call("toggle", className)
}

func (o *Object) ScrollHeight() int {
	scrollHeight := o.value.Get("scrollHeight")
	if scrollHeight.Type() == js.TypeNumber {
		return scrollHeight.Int()
	}

	return 0
}

func (o *Object) Focus() {
	o.value.Call("focus")
}

func (o *Object) ScrollIntoView() {
	o.value.Call("scrollIntoView")
}

func (o *Object) Style() *Style {
	return &Style{
		value: o.value.Get("style"),
	}
}

type Style struct {
	value js.Value
}

func (s *Style) Width(value string) {
	s.value.Set("width", value)
}

func (s *Style) Height(value string) {
	s.value.Set("height", value)
}

func (o *Object) Location() *Location {
	l := o.value.Get("location")

	return &Location{
		Protocol: l.Get("protocol").String(),
		Host:     l.Get("host").String(),
		Hostname: l.Get("hostname").String(),
		Port:     l.Get("port").String(),
		Pathname: l.Get("pathname").String(),
	}
}

type Location struct {
	Protocol string
	Host     string
	Hostname string
	Port     string
	Pathname string
}
