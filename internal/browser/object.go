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

func Confirm(text string) bool {
	return Global().value.Call("confirm", text).Bool()
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

func (o *Object) GetTextContent() string {
	return o.value.Get("textContent").String()
}

func (o *Object) TextContent(text string) {
	o.value.Set("textContent", text)
}

func (o *Object) GetValue() string {
	return o.value.Get("value").String()
}

func (o *Object) SetValue(value string) {
	o.value.Set("value", value)
}

func (o *Object) GetElementByID(id string) *Object {
	return &Object{
		value: o.value.Call("getElementById", id),
	}
}

func (o *Object) GetElementsByClassName(class string) []*Object {
	elements := o.value.Call("getElementsByClassName", class)

	length := elements.Length()
	var result []*Object

	for i := 0; i < length; i++ {
		result = append(
			result,
			&Object{
				value: elements.Index(i),
			},
		)
	}

	return result
}

func (o *Object) GetElementsByTagName(tag string) []*Object {
	elements := o.value.Call("getElementsByTagName", tag)

	length := elements.Length()
	var result []*Object

	for i := 0; i < length; i++ {
		result = append(
			result,
			&Object{
				value: elements.Index(i),
			},
		)
	}

	return result
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

func (o *Object) ClosestByClassName(className string) *Object {
	result := o

	for !result.ContainsClass(className) {
		value := result.value.Get("parentElement")
		if value.IsNull() {
			return nil
		}
		result = &Object{
			value: value,
		}
	}

	return result
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

func (o *Object) Disabled(disabled bool) {
	o.value.Set("disabled", disabled)
}

func (o *Object) RemoveChildren() {
	o.value.Call("replaceChildren")
}

func (o *Object) Append(children ...*Object) {
	for _, c := range children {
		o.value.Call("appendChild", c.value)
	}
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

func (o *Object) RemoveClass(className string) {
	o.value.Get("classList").Call("remove", className)
}

func (o *Object) ToggleClass(className string) {
	o.value.Get("classList").Call("toggle", className)
}

func (o *Object) ContainsClass(className string) bool {
	result := o.value.Get("classList").Call("contains", className)
	if result.Type() == js.TypeBoolean {
		return result.Bool()
	}

	return false
}

func (o *Object) ScrollHeight() int {
	scrollHeight := o.value.Get("scrollHeight")
	if scrollHeight.Type() == js.TypeNumber {
		return scrollHeight.Int()
	}

	return 0
}

func (o *Object) Blur() {
	o.value.Call("blur")
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
		value:    l,
	}
}

type Location struct {
	Protocol string
	Host     string
	Hostname string
	Port     string
	Pathname string
	value    js.Value
}

func (l *Location) Reload() {
	l.value.Call("reload")
}
