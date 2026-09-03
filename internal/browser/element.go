package browser

import (
	"strings"
)

const (
	InputTypeText     = "text"
	InputTypePassword = "password"
	InputTypeCheckbox = "checkbox"
)

func element(elementType string, classes []string, children []*Object) *Object {
	e := Document().value.Call("createElement", elementType)

	if len(classes) > 0 {
		e.Call("setAttribute", "class", strings.Join(classes, " "))
	}

	for _, c := range children {
		e.Call("appendChild", c.value)
	}

	return &Object{
		value: e,
	}
}

func Div(classes []string, children ...*Object) *Object {
	return element("div", classes, children)
}

func Span(classes []string, children ...*Object) *Object {
	return element("span", classes, children)
}

func H3(children ...*Object) *Object {
	return element("h3", nil, children)
}

func H4(children ...*Object) *Object {
	return element("h4", nil, children)
}

func P(children ...*Object) *Object {
	return element("p", nil, children)
}

func Strong(children ...*Object) *Object {
	return element("strong", nil, children)
}

func Em(children ...*Object) *Object {
	return element("em", nil, children)
}

func UL(children ...*Object) *Object {
	return element("ul", nil, children)
}

func LI(children ...*Object) *Object {
	return element("li", nil, children)
}

func Input(classes []string, inputType, value string) *Object {
	o := element("input", classes, nil)
	o.value.Call("setAttribute", "type", inputType)
	o.value.Call("setAttribute", "value", value)
	return o
}

func Label(classes []string, children ...*Object) *Object {
	return element("label", nil, children)
}

func Button(classes []string, children ...*Object) *Object {
	o := element("button", classes, children)
	o.value.Call("setAttribute", "type", "button")
	return o
}

func Text(data string) *Object {
	e := Document().value.Call("createTextNode", data)

	return &Object{
		value: e,
	}
}

func HTML(html string) *Object {
	r := Document().value.Call("createRange")
	e := r.Call("createContextualFragment", html)

	return &Object{
		value: e,
	}
}
