package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	b "github.com/xi0/coderoom-ai/internal/browser"
	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// CSS classes used to hide a message at a given verbosity level. The active
// level is set as a class on the document element (see applyVerbosity in
// ui.go), and these classes are matched by styles.css.
const (
	verbosityHideMinimal = "verbosity-hide-minimal"
	verbosityHideMedium  = "verbosity-hide-medium"
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

// lastMessage holds the most recently added message. It is used to collapse
// consecutive identical tool messages into a single message that carries a
// counter pill.
var lastMessage *b.Object

func addMessage(m *b.Object) {
	// A nil message means that the caller decided not to add a new message
	// (e.g. a repeated tool call that was merged into the previous message).
	if m == nil {
		return
	}

	doc := b.Document()
	workingMessage := doc.GetElementByID("working-message")
	parent := workingMessage.Parent()

	parent.InsertBefore(m, workingMessage)

	lastMessage = m

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

// reasoningMessage renders the model's reasoning content ("reasoning_content").
// It is only visible at the high verbosity level, so it carries both the
// minimal and medium "verbosity-hide" classes.
func reasoningMessage(markdown string) *b.Object {
	return b.Div(
		[]string{"message", "system-message", "reasoning-message", verbosityHideMinimal, verbosityHideMedium},
		b.Div(
			[]string{"message-content"},
			b.Div(
				[]string{"reasoning-header"},
				b.Span(
					[]string{"reasoning-icon"},
					b.Text("💭"),
				),
				b.Span(
					[]string{"reasoning-label"},
					b.Text("Reasoning"),
				),
			),
			b.Div(
				[]string{"reasoning-text"},
				b.HTML(renderMarkdown(markdown)),
			),
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

func toolMessage(message *wire.ToolMessage) *b.Object {
	// When a tool message provides a MultipleFmt, consecutive calls to the same
	// tool are collapsed into the previous message by showing a counter pill
	// instead of adding a new message.
	if message.MultipleFmt != nil &&
		lastMessage != nil &&
		lastMessage.ContainsClass("tool-message") &&
		toolMessageString(lastMessage) == message.Tool {
		addToolMessageCount(lastMessage, *message.MultipleFmt)
		return nil
	}

	// For tools without a MultipleFmt, consecutive calls to the same tool
	// (identified by the text before the first "(") are appended to the
	// previous message on separate lines instead of creating a new message.
	if message.MultipleFmt == nil &&
		lastMessage != nil &&
		lastMessage.ContainsClass("tool-message-lines") &&
		toolName(toolMessageString(lastMessage)) == toolName(message.Tool) {
		addToolMessageLine(lastMessage, message.Tool)
		return nil
	}

	toolIcon := b.Span(
		[]string{"tool-icon"},
		b.Text("🔧"),
	)

	functionCall := b.Span(
		[]string{"function-call"},
		b.Text(message.Tool),
	)

	// Tools without a MultipleFmt group repeated calls on separate lines. The
	// "Tool:" label lives in its own column (see the .tool-message-lines rules
	// in styles.css) and each call is a sibling of it, so that the calls all
	// line up with each other rather than each line starting at the left edge.
	if message.MultipleFmt == nil {
		return b.Div(
			[]string{"message", "system-message", "tool-message", "tool-message-lines", verbosityHideMinimal},
			b.Div(
				[]string{"message-content"},
				toolIcon,
				b.Span(
					[]string{"message-text"},
					b.Text("Tool:"),
				),
				functionCall,
			),
		)
	}

	return b.Div(
		[]string{"message", "system-message", "tool-message", verbosityHideMinimal},
		b.Div(
			[]string{"message-content"},
			toolIcon,
			b.Span(
				[]string{"message-text"},
				b.Text("Tool: "),
				functionCall,
			),
		),
	)

}

// toolName returns the part of a tool call before the first "(". It is used to
// group repeated calls to the same tool.
func toolName(tool string) string {
	if idx := strings.Index(tool, "("); idx >= 0 {
		return tool[:idx]
	}

	return tool
}

// toolMessageString returns the tool string displayed in a tool message.
func toolMessageString(message *b.Object) string {
	functionCalls := message.GetElementsByClassName("function-call")
	if len(functionCalls) == 0 {
		return ""
	}

	return functionCalls[0].GetTextContent()
}

// addToolMessageLine appends another tool call to an existing tool message,
// placing it on its own line (each call is a separate grid item; see
// .tool-message-lines in styles.css).
func addToolMessageLine(message *b.Object, tool string) {
	contents := message.GetElementsByClassName("message-content")
	if len(contents) == 0 {
		return
	}

	contents[0].Append(
		b.Span(
			[]string{"function-call"},
			b.Text(tool),
		),
	)
}

// addToolMessageCount shows or increments the counter pill in a tool message.
// The format string is expected to contain a single %d placeholder which is
// replaced with the number of collapsed tool calls.
func addToolMessageCount(message *b.Object, format string) {
	pills := message.GetElementsByClassName("tool-count-pill")

	var pill *b.Object
	count := 2
	if len(pills) > 0 {
		pill = pills[0]

		count = 1
		if parsed, err := strconv.Atoi(pill.GetAttribute("data-count")); err == nil {
			count = parsed
		}
		count++
	} else {
		pill = b.Span([]string{"tool-count-pill"})

		texts := message.GetElementsByClassName("message-text")
		if len(texts) > 0 {
			texts[0].Append(pill)
		} else {
			message.GetElementsByClassName("message-content")[0].Append(pill)
		}
	}

	pill.SetAttribute("data-count", strconv.Itoa(count))
	pill.TextContent(fmt.Sprintf(format, count))
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

// filesMessage renders the list of files that were changed during the session.
// Only the most recent files message is kept: any previous one in the
// conversation is removed so that the displayed list is never stale.
func filesMessage(message *wire.FilesMessage) *b.Object {
	doc := b.Document()
	for _, previous := range doc.GetElementsByClassName("files-message") {
		previous.Remove()
	}

	var fileItems []*b.Object
	for _, file := range message.Files {
		fileItems = append(fileItems, fileMessageItem(file))
	}

	return b.Div(
		[]string{"message", "system-message", "files-message"},
		b.Div(
			[]string{"message-content"},
			b.P(
				b.Text("The following files have been changed during this session:"),
			),
			b.UL(
				fileItems...,
			),
		),
	)
}

// fileMessageItem renders a single file entry of a files message. Files that
// were written in full or modified get a button that opens a dialog showing
// either the entire file or the diff; the action taken on the file is shown
// next to its name in every case.
func fileMessageItem(file wire.File) *b.Object {
	switch file.Action {
	case wire.FileActionEntireFile, wire.FileActionModifiedFile:
		button := b.Button(
			[]string{"file-btn"},
			b.Text(file.Filename),
		)
		button.AddClickHandler(func(this, e *b.Object) any {
			e.PreventDefault()

			fileDialog(file)

			return nil
		})

		return b.LI(
			button,
			b.Span(
				[]string{"file-action"},
				b.Text(" "+fileActionLabel(file.Action)),
			),
		)
	default:
		return b.LI(
			b.Text(fmt.Sprintf("%s (%s)", file.Filename, fileActionLabel(file.Action))),
		)
	}
}

// fileActionLabel returns the human-readable label for a file action.
func fileActionLabel(action string) string {
	switch action {
	case wire.FileActionEntireFile:
		return "Entire file"
	case wire.FileActionModifiedFile:
		return "Modified"
	case wire.FileActionDeletedFile:
		return "Deleted"
	default:
		return action
	}
}

// fileDialog opens a modal dialog that displays the content of a changed file.
// The content is fetched asynchronously from the file's URI; for an
// "entire_file" action it holds the whole file and for a "modified_file"
// action it holds the diff.
func fileDialog(file wire.File) {
	dialog := b.Dialog([]string{"file-dialog"})
	dialog.SetAttribute("closedby", "any")

	content := b.Pre(
		[]string{"file-content"},
		b.Text("Loading..."),
	)

	closeButton := b.Button(
		[]string{"secondary-btn"},
		b.Text("Close"),
	)
	closeButton.AddClickHandler(func(this, e *b.Object) any {
		e.PreventDefault()
		dialog.Close()
		return nil
	})

	dialog.Append(
		b.H2(
			b.Text(file.Filename),
		),
		content,
		b.Div(
			[]string{"file-dialog-actions"},
			closeButton,
		),
	)

	// The dialog is attached to the document body rather than to the message so
	// that it is not removed when a newer files message replaces the current
	// one. It removes itself once closed.
	dialog.AddEventHandler("close", func(this, e *b.Object) any {
		dialog.Remove()
		return nil
	})

	doc := b.Document()
	doc.Body().Append(dialog)

	dialog.ShowModal()

	go loadFileContent(file, content)
}

// loadFileContent fetches the content of a changed file from its URI and
// displays it in the given element.
func loadFileContent(file wire.File, content *b.Object) {
	location := b.Document().Location()
	url := fmt.Sprintf("%s//%s%s", location.Protocol, location.Host, file.URI)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		fmt.Printf("Get(): %v\n", err)
		content.TextContent(fmt.Sprintf("Unable to load file: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content.TextContent(fmt.Sprintf("Unable to load file: %s", resp.Status))
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("io.ReadAll(): %v\n", err)
		content.TextContent(fmt.Sprintf("Unable to read file: %v", err))
		return
	}

	var edit wire.EditResponse
	if err := json.Unmarshal(data, &edit); err != nil {
		fmt.Printf("json.Unmarshal(): %v\n", err)
		content.TextContent(fmt.Sprintf("Unable to parse file: %v\n", err))
		return
	}

	// For a modified file the content is a unified diff, which is colorized so
	// that additions, deletions and hunk headers stand out. Any other content
	// (e.g. the entire file) is shown as plain text.
	if file.Action == wire.FileActionModifiedFile {
		colorizeDiff(content, edit.Content)
	} else {
		content.TextContent(edit.Content)
	}
}

// colorizeDiff renders a unified diff as a sequence of colorized lines inside
// the given element. Each line is wrapped in a span whose class encodes the
// kind of line so that it can be styled via CSS.
func colorizeDiff(content *b.Object, diff string) {
	content.RemoveChildren()

	// A trailing newline would produce a final, empty line.
	diff = strings.TrimSuffix(diff, "\n")

	for _, line := range strings.Split(diff, "\n") {
		content.Append(
			b.Span(
				[]string{"diff-line", diffLineClass(line)},
				b.Text(line),
			),
		)
	}
}

// diffLineClass classifies a single line of a unified diff based on its prefix
// so that it can be colorized. The file and hunk header checks come first
// because their prefixes ("+++", "---", "@@") also match the addition and
// deletion checks.
func diffLineClass(line string) string {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return "diff-file"
	case strings.HasPrefix(line, "@@"):
		return "diff-hunk"
	case strings.HasPrefix(line, "+"):
		return "diff-add"
	case strings.HasPrefix(line, "-"):
		return "diff-del"
	default:
		return "diff-context"
	}
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
