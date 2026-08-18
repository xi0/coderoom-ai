package ui

import (
	"fmt"

	b "github.com/xi0/coderoom-ai/internal/browser"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func renderMarkdown(md string) string {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	doc := p.Parse([]byte(md))

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if heading, ok := node.(*ast.Heading); ok {
			heading.Level = 4
		}
		return ast.GoToNext
	})

	renderer := html.NewRenderer(html.RendererOptions{
		Flags: html.CommonFlags,
	})
	return string(markdown.Render(doc, renderer))
}

func addMessage(m *b.Object) {
	doc := b.Document()
	workingMessage := doc.GetElementByID("working-message")
	parent := workingMessage.Parent()

	parent.InsertBefore(m, workingMessage)

	workingMessage.ScrollIntoView()
}

func systemMessage(markdown string) *b.Object {
	return b.Div(
		[]string{"message", "system-message"},
		b.Div(
			[]string{"message-content"},
			b.HTML(renderMarkdown(markdown)),
		),
	)
}

func userMessage(markdown string) *b.Object {
	return b.Div(
		[]string{"message", "user-message"},
		b.Div(
			[]string{"message-content"},
			b.HTML(renderMarkdown(markdown)),
		),
	)
}

func toolMessage(tool string) *b.Object {
	return b.Div(
		[]string{"message", "system-message"},
		b.Div(
			[]string{"message-content"},
			b.Span(
				[]string{"tool-icon"},
				b.Text("🔧"),
			),
			b.Span(
				[]string{"message-text"},
				b.Text("Tool: "),
				b.Span(
					[]string{"function-call"},
					b.Text(tool),
				),
			),
		),
	)

}

func proposalMessage(markdown string) *b.Object {
	return b.Div(
		[]string{"message", "system-message", "plan-message"},
		b.Div(
			[]string{"message-content"},
			b.H3(
				b.Text("📋 Proposed Plan"),
			),
			b.HTML(renderMarkdown(markdown)),
			b.P(
				b.Em(
					b.Text("Please review this plan and decide whether to proceed or decline."),
				),
			),
			b.Div(
				[]string{"plan-actions"},
				b.Button(
					[]string{"plan-btn", "plan-btn-decline"},
					b.Text("Decline"),
				),
				b.Button(
					[]string{"plan-btn", "plan-btn-proceed"},
					b.Text("Proceed"),
				),
			),
		),
	)
}

func optionsMessage(options []string) *b.Object {
	var buttons []*b.Object

	for i, o := range options {
		button := b.Button(
			[]string{"option-btn"},
			b.Text(o),
		)
		button.SetAttribute("data-option", fmt.Sprintf("%d", i))

		buttons = append(buttons, button)
	}

	return b.Div(
		[]string{"message", "system-message", "options-message"},
		b.Div(
			[]string{"message-content"},
			b.H3(
				b.Text("☰ Please select an option:"),
			),
			b.Div(
				[]string{"options-container"},
				buttons...,
			),
			b.Div(
				[]string{"options-actions"},
				b.Button(
					[]string{"options-btn-proceed"},
					b.Text("Proceed"),
				),
			),
		),
	)

}
