package ui

import (
	"fmt"
	"strconv"

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

func allowDirMessage(dir string, confirm func(bool)) *b.Object {
	denyButton := b.Button(
		[]string{"plan-btn", "allow-btn-deny"},
		b.Text("Deny"),
	)

	denyButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		allowDirMessageDone(this, false)
		go confirm(false)
		return nil
	})

	allowButton := b.Button(
		[]string{"plan-btn", "allow-btn-allow"},
		b.Text("Allow"),
	)

	allowButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		allowDirMessageDone(this, true)
		go confirm(true)
		return nil
	})

	return b.Div(
		[]string{"message", "system-message", "allow-dir-message"},
		b.Div(
			[]string{"message-content"},
			b.P(
				b.Text(fmt.Sprintf("Do you want to allow CoderoomAI to work in the directory %q?", dir)),
			),
			b.P(
				b.Text("Coderoom AI might read and modify files under this directory and execute the configured tools."),
			),
			b.P(
				b.Text("Although the tool does have revert functionality, it is stronhly encouraged to use a version control system (e.g. git) for checkpointing the code."),
			),
			b.P(
				b.Em(
					b.Text("Please consider this question and decide whether to allow or deny."),
				),
			),
			b.Div(
				[]string{"allow-dir-actions"},
				denyButton,
				allowButton,
			),
		),
	)
}

func allowDirMessageDone(button *b.Object, confirmed bool) {
	var text string
	if confirmed {
		text = "Directory was allowed"
	} else {
		text = "Directory was denied"
	}

	message := button.ClosestByClassName("allow-dir-message")

	actions := message.GetElementsByClassName("allow-dir-actions")[0]
	buttons := actions.GetElementsByTagName("button")
	for _, b := range buttons {
		b.Disabled(true)
	}

	actions.RemoveChildren()
	actions.Append(b.P(
		b.Text(text),
	))
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

func proposalMessage(markdown string, confirm func(bool)) *b.Object {
	declineButton := b.Button(
		[]string{"plan-btn", "plan-btn-decline"},
		b.Text("Decline"),
	)

	declineButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		proposalMessageDone(this, false)
		go confirm(false)
		return nil
	})

	proceedButton := b.Button(
		[]string{"plan-btn", "plan-btn-proceed"},
		b.Text("Proceed"),
	)

	proceedButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		proposalMessageDone(this, true)
		go confirm(true)
		return nil
	})

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
				declineButton,
				proceedButton,
			),
		),
	)
}

func proposalMessageDone(button *b.Object, confirmed bool) {
	var text string
	if confirmed {
		text = "Proposal was accepted"
	} else {
		text = "Proposal was rejected"
	}

	doc := b.Document()
	message := button.ClosestByClassName("plan-message")

	actions := message.GetElementsByClassName("plan-actions")[0]
	buttons := actions.GetElementsByTagName("button")
	for _, b := range buttons {
		b.Disabled(true)
	}

	actions.RemoveChildren()
	actions.Append(b.P(
		b.Text(text),
	))

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", 0))

	workingMessage := doc.GetElementByID("working-message")
	textElement := workingMessage.GetElementsByClassName("message-text")[0]
	textElement.TextContent("Working...")

	workingMessage.RemoveClass("hidden")
	workingMessage.ScrollIntoView()
}

func blockedMessage(action string, filenames []string, confirm func(bool)) *b.Object {
	denyButton := b.Button(
		[]string{"plan-btn", "blocked-btn-deny"},
		b.Text("Deny"),
	)

	denyButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		blockedMessageDone(this, false)
		go confirm(false)
		return nil
	})

	allowButton := b.Button(
		[]string{"plan-btn", "blocked-btn-allow"},
		b.Text("Allow"),
	)

	allowButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		blockedMessageDone(this, true)
		go confirm(true)
		return nil
	})

	var fileItems []*b.Object
	for _, filename := range filenames {
		fileItems = append(fileItems, b.LI(
			b.Text(filename),
		))
	}

	return b.Div(
		[]string{"message", "system-message", "blocked-message"},
		b.Div(
			[]string{"message-content"},
			b.H3(
				b.Text("Action blocked"),
			),
			b.P(
				b.Text(fmt.Sprintf("The action %q has been blocked, because the following files were modified during this session:", action)),
			),
			b.UL(
				fileItems...,
			),
			b.P(
				b.Em(
					b.Text("Please review the changes and decide whether to allow or deny the action"),
				),
			),
			b.Div(
				[]string{"blocked-actions"},
				denyButton,
				allowButton,
			),
		),
	)
}

func blockedMessageDone(button *b.Object, confirmed bool) {
	var text string
	if confirmed {
		text = "Action was allowed"
	} else {
		text = "Action was denied"
	}

	doc := b.Document()
	message := button.ClosestByClassName("blocked-message")

	actions := message.GetElementsByClassName("blocked-actions")[0]
	buttons := actions.GetElementsByTagName("button")
	for _, b := range buttons {
		b.Disabled(true)
	}

	actions.RemoveChildren()
	actions.Append(b.P(
		b.Text(text),
	))

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", 0))

	workingMessage := doc.GetElementByID("working-message")
	textElement := workingMessage.GetElementsByClassName("message-text")[0]
	textElement.TextContent("Working...")

	workingMessage.RemoveClass("hidden")
	workingMessage.ScrollIntoView()
}

func optionsMessage(description string, options []string, sendOption func(int)) *b.Object {
	buttons := []*b.Object{
		b.P(
			b.Text(description),
		),
	}

	for i, o := range options {
		button := b.Button(
			[]string{"option-btn"},
			b.Text(o),
		)
		button.SetAttribute("data-option", fmt.Sprintf("%d", i))
		button.AddClickHandler(optionClicked)

		buttons = append(buttons, button)
	}

	proceedButton := b.Button(
		[]string{"options-btn-proceed"},
		b.Text("Proceed"),
	)
	proceedButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		option := optionsProceed(this)
		if option != -1 {
			go sendOption(option)
		}
		return nil
	})

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
				proceedButton,
			),
		),
	)
}

func optionClicked(this, e *b.Object) any {
	e.PreventDefault()

	parent := this.Parent()
	buttons := parent.GetElementsByTagName("button")
	for _, b := range buttons {
		b.RemoveClass("selected")
	}

	this.AddClass("selected")

	return nil
}

func optionsProceed(this *b.Object) int {

	message := this.ClosestByClassName("options-message")
	buttons := message.GetElementsByClassName("option-btn")

	var option string
	var optionText string
	for _, b := range buttons {
		if b.ContainsClass("selected") {
			option = b.GetAttribute("data-option")
			optionText = b.GetTextContent()
		}
	}

	optionParsed := -1
	var err error
	optionParsed, err = strconv.Atoi(option)
	if err != nil {
		fmt.Printf("%s could not be parsed as an integer", option)
	}

	if option == "" || optionParsed == -1 {
		b.Alert("No option selected")
		return -1
	}

	for _, b := range buttons {
		b.Disabled(true)
	}
	this.Disabled(true)

	actions := message.GetElementsByClassName("options-actions")[0]
	actions.RemoveChildren()
	actions.Append(b.P(
		b.Text(fmt.Sprintf("%q was selected", optionText)),
	))

	doc := b.Document()

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", 0))

	workingMessage := doc.GetElementByID("working-message")
	textElement := workingMessage.GetElementsByClassName("message-text")[0]
	textElement.TextContent("Working...")

	workingMessage.RemoveClass("hidden")
	workingMessage.ScrollIntoView()

	return optionParsed
}
