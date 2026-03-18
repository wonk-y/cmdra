package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	agentv1 "cmdra/gen/agent/v1"
	"cmdra/internal/tui/attachterm"
	"cmdra/pkg/cmdraclient"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type section int

const (
	sectionExecutions section = iota
	sectionTransfers
	sectionNewCommand
	sectionNewTransfer
	sectionConnection
)

type commandMode int

const (
	commandModeArgv commandMode = iota
	commandModeShell
	commandModeSession
)

type transferMode int

const (
	transferModeUpload transferMode = iota
	transferModeDownload
	transferModeArchive
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusList
	focusDetail
	focusForm
)

type styles struct {
	doc          lipgloss.Style
	sidebar      lipgloss.Style
	navItem      lipgloss.Style
	navActive    lipgloss.Style
	title        lipgloss.Style
	subtitle     lipgloss.Style
	border       lipgloss.Style
	panelTitle   lipgloss.Style
	selectedRow  lipgloss.Style
	muted        lipgloss.Style
	error        lipgloss.Style
	success      lipgloss.Style
	status       lipgloss.Style
	statusValue  lipgloss.Style
	footer       lipgloss.Style
	chip         lipgloss.Style
	chipActive   lipgloss.Style
	attachBanner lipgloss.Style
}

func newStyles() styles {
	return styles{
		doc:         lipgloss.NewStyle().Padding(1, 2),
		sidebar:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1),
		navItem:     lipgloss.NewStyle().Padding(0, 1),
		navActive:   lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")),
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		subtitle:    lipgloss.NewStyle().Foreground(lipgloss.Color("110")),
		border:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1),
		panelTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
		selectedRow: lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60")),
		muted:       lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		error:       lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		success:     lipgloss.NewStyle().Foreground(lipgloss.Color("84")).Bold(true),
		status:      lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		statusValue: lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		footer:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		chip:        lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("248")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		chipActive:  lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		attachBanner: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("57")).
			Padding(0, 1),
	}
}

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Refresh     key.Binding
	Back        key.Binding
	Quit        key.Binding
	NextFocus   key.Binding
	PrevFocus   key.Binding
	ToggleHelp  key.Binding
	Cancel      key.Binding
	Attach      key.Binding
	Delete      key.Binding
	ClearAll    key.Binding
	ViewOutput  key.Binding
	WriteStdin  key.Binding
	SendEOF     key.Binding
	RunningOnly key.Binding
	Submit      key.Binding
	NextMode    key.Binding
	PrevMode    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear error")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		NextFocus:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field/panel")),
		PrevFocus:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field/panel")),
		ToggleHelp:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		Cancel:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cancel")),
		Attach:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "attach")),
		Delete:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete item")),
		ClearAll:    key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "clear history")),
		ViewOutput:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "toggle output")),
		WriteStdin:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "send stdin line")),
		SendEOF:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "send EOF")),
		RunningOnly: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "running only")),
		Submit:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		NextMode:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next mode")),
		PrevMode:    key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev mode")),
	}
}

type app struct {
	client *cmdraclient.Client
	conn   cmdraclient.DialConfig

	width  int
	height int
	ready  bool

	styles   styles
	keys     keyMap
	help     help.Model
	spinner  spinner.Model
	debugPTY bool

	section section
	focus   focusArea

	executions []*agentv1.Execution
	transfers  []*agentv1.Execution
	selection  map[section]int
	selectedID map[section]string

	detailViewport viewport.Model
	detailMeta     *agentv1.Execution
	detailOutput   []string
	showOutput     bool
	runningOnly    bool
	detailKey      string

	commandMode     commandMode
	transferMode    transferMode
	commandInputs   []textinput.Model
	transferInputs  []textinput.Model
	connectInputs   []textinput.Model
	detailInput     textinput.Model
	detailInputOpen bool
	formCursor      int
	lastSubmittedID string
	pendingAction   destructiveAction
	pendingTargetID string

	attach *attachState

	status     string
	err        error
	loading    bool
	showHelp   bool
	statusWhen time.Time
}

type attachState struct {
	executionID string
	usesPTY     bool
	session     attachSession
	cancel      context.CancelFunc
	viewport    viewport.Model
	terminal    *attachterm.Model
	transcript  strings.Builder
	awaitingCmd bool
	exited      bool
	closeOnExit bool
}

type attachSession interface {
	Recv() (*agentv1.AttachEvent, error)
	SendStdin(data []byte, eof bool) error
	CancelExecution() error
	ResizePTY(rows, cols uint32) error
	CloseSend() error
}

type destructiveAction int

const (
	destructiveActionNone destructiveAction = iota
	destructiveActionDelete
	destructiveActionClearHistory
)

type loadExecutionsMsg struct {
	items []*agentv1.Execution
	err   error
}

type loadDetailMsg struct {
	exec   *agentv1.Execution
	output []string
	err    error
}

type startExecutionMsg struct {
	exec *agentv1.Execution
	err  error
}

type uploadDoneMsg struct {
	resp *agentv1.UploadFileResponse
	err  error
}

type downloadDoneMsg struct {
	localPath string
	remote    []string
	resp      *cmdraclient.DownloadResult
	err       error
}

type attachConnectedMsg struct {
	session attachSession
	cancel  context.CancelFunc
	ack     *agentv1.Execution
	err     error
}

type attachEventMsg struct {
	event *agentv1.AttachEvent
	err   error
}

type tickMsg time.Time

type historyDeletedMsg struct {
	executionID string
	err         error
}

type historyClearedMsg struct {
	deletedCount        uint64
	skippedRunningCount uint64
	err                 error
}

type stdinWrittenMsg struct {
	executionID string
	bytesSent   int
	eof         bool
	err         error
}

func New(client *cmdraclient.Client, cfg cmdraclient.DialConfig) tea.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line
	detail := viewport.New(0, 0)
	detail.SetContent("Loading…")

	commandInputs := make([]textinput.Model, 3)
	commandInputs = make([]textinput.Model, 4)
	for i := range commandInputs {
		commandInputs[i] = textinput.New()
		commandInputs[i].CharLimit = 4096
		commandInputs[i].Width = 80
	}
	commandInputs[0].Placeholder = "Binary or shell path"
	commandInputs[1].Placeholder = "Args or command string"
	commandInputs[2].Placeholder = "Shell args (session mode only)"
	commandInputs[3].Placeholder = "Use PTY (true/false)"
	commandInputs[0].Focus()

	transferInputs := make([]textinput.Model, 3)
	for i := range transferInputs {
		transferInputs[i] = textinput.New()
		transferInputs[i].CharLimit = 4096
		transferInputs[i].Width = 80
	}
	transferInputs[0].Placeholder = "Local path"
	transferInputs[1].Placeholder = "Remote path(s)"
	transferInputs[2].Placeholder = "Chunk size (optional, default 32768)"

	connectInputs := make([]textinput.Model, 5)
	for i := range connectInputs {
		connectInputs[i] = textinput.New()
		connectInputs[i].Width = 80
		connectInputs[i].CharLimit = 4096
	}
	connectInputs[0].SetValue(cfg.Address)
	connectInputs[1].SetValue(cfg.CAFile)
	connectInputs[2].SetValue(cfg.ClientCertFile)
	connectInputs[3].SetValue(cfg.ClientKeyFile)
	connectInputs[4].SetValue(cfg.ServerName)
	connectInputs[0].Placeholder = "Address"
	connectInputs[1].Placeholder = "CA PEM"
	connectInputs[2].Placeholder = "Client cert PEM"
	connectInputs[3].Placeholder = "Client key PEM"
	connectInputs[4].Placeholder = "Server name override"

	detailInput := textinput.New()
	detailInput.CharLimit = 4096
	detailInput.Width = 80
	detailInput.Placeholder = "Type one stdin line and press enter"

	h := help.New()
	h.ShowAll = false

	return &app{
		client:         client,
		conn:           cfg,
		styles:         newStyles(),
		keys:           newKeyMap(),
		help:           h,
		spinner:        sp,
		debugPTY:       os.Getenv("CMDRAUI_PTY_DEBUG") != "",
		section:        sectionExecutions,
		focus:          focusList,
		selection:      map[section]int{},
		selectedID:     map[section]string{},
		detailViewport: detail,
		showOutput:     true,
		commandMode:    commandModeArgv,
		transferMode:   transferModeUpload,
		commandInputs:  commandInputs,
		transferInputs: transferInputs,
		connectInputs:  connectInputs,
		detailInput:    detailInput,
		status:         "Connecting…",
		loading:        true,
	}
}

func (a *app) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, a.refreshCmd(), tickCmd(), literalMsgCmd(tea.EnableBracketedPaste()))
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.resize()
		if a.attach != nil && a.attach.usesPTY {
			rows, cols := a.attachPTYSize()
			return a, attachResizeCmd(a.attach.session, rows, cols)
		}
		return a, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd
	case tickMsg:
		if a.attach == nil {
			cmds = append(cmds, a.refreshCmd())
		}
		return a, tea.Batch(append(cmds, tickCmd())...)
	case loadExecutionsMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		previousExecutionID := a.selectedID[sectionExecutions]
		if previousExecutionID == "" {
			if item := selectedFromList(a.executions, a.selection[sectionExecutions]); item != nil {
				previousExecutionID = primarySelectionID(item)
			}
		}
		previousTransferID := a.selectedID[sectionTransfers]
		if previousTransferID == "" {
			if item := selectedFromList(a.transfers, a.selection[sectionTransfers]); item != nil {
				previousTransferID = primarySelectionID(item)
			}
		}
		a.executions, a.transfers = splitExecutions(msg.items)
		a.selectedID[sectionExecutions] = previousExecutionID
		a.selectedID[sectionTransfers] = previousTransferID
		a.clampSelections()
		cmds = append(cmds, a.loadDetailForSelection())
		return a, tea.Batch(cmds...)
	case loadDetailMsg:
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.detailMeta = msg.exec
		a.detailOutput = msg.output
		a.syncDetailViewport()
		return a, nil
	case startExecutionMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.lastSubmittedID = msg.exec.GetExecutionId()
		a.section = sectionExecutions
		a.focus = focusSidebar
		a.setStatus(fmt.Sprintf("Started %s", msg.exec.GetExecutionId()))
		return a, a.refreshCmd()
	case uploadDoneMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.section = sectionTransfers
		a.focus = focusSidebar
		a.lastSubmittedID = msg.resp.GetTransferId()
		a.setStatus(fmt.Sprintf("Uploaded %s (%s)", msg.resp.GetPath(), msg.resp.GetTransferId()))
		return a, a.refreshCmd()
	case downloadDoneMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.section = sectionTransfers
		a.focus = focusSidebar
		a.lastSubmittedID = msg.resp.TransferID
		a.setStatus(fmt.Sprintf("Saved %s (%s)", msg.localPath, msg.resp.TransferID))
		return a, a.refreshCmd()
	case attachConnectedMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		vp := viewport.New(a.width-4, max(1, a.height-6))
		a.attach = &attachState{executionID: msg.ack.GetExecutionId(), usesPTY: msg.ack.GetUsesPty(), session: msg.session, cancel: msg.cancel, viewport: vp}
		if a.attach.usesPTY {
			a.attach.terminal = attachterm.New(max(1, a.width-4), max(1, a.height-4))
		} else {
			a.attach.transcript.WriteString(renderExecutionSummary(msg.ack))
			a.attach.transcript.WriteString("\n")
			a.attach.viewport.SetContent(a.attach.transcript.String())
		}
		a.setStatus("Attach mode active. Use ctrl+g q to detach, ctrl+g c to cancel, ctrl+g h for help.")
		cmds = append(cmds, a.recvAttachCmd())
		if a.attach.usesPTY {
			rows, cols := a.attachPTYSize()
			cmds = append(cmds, attachResizeCmd(a.attach.session, rows, cols))
		}
		return a, tea.Batch(cmds...)
	case attachEventMsg:
		if a.attach == nil {
			return a, nil
		}
		if msg.err != nil {
			if errors.Is(msg.err, context.Canceled) {
				return a, nil
			}
			a.setError(msg.err)
			a.detach()
			return a, nil
		}
		switch payload := msg.event.GetPayload().(type) {
		case *agentv1.AttachEvent_Output:
			if a.attach.usesPTY && a.attach.terminal != nil {
				a.attach.terminal.Write(payload.Output.GetData())
			} else {
				a.attach.transcript.Write(payload.Output.GetData())
				a.attach.viewport.SetContent(a.attach.transcript.String())
				a.attach.viewport.GotoBottom()
			}
			return a, a.recvAttachCmd()
		case *agentv1.AttachEvent_Exit:
			if a.attach.usesPTY && a.attach.terminal != nil {
				if a.attach.closeOnExit {
					a.detach()
					a.setStatus("Attached execution canceled and closed.")
					return a, a.refreshCmd()
				}
				a.setStatus("Execution exited. Press ctrl+g q to leave attach mode.")
			} else {
				a.attach.transcript.WriteString("\nExecution exited:\n")
				a.attach.transcript.WriteString(renderExecutionSummary(payload.Exit.GetExecution()))
				a.attach.viewport.SetContent(a.attach.transcript.String())
				a.attach.viewport.GotoBottom()
				if a.attach.closeOnExit {
					a.detach()
					a.setStatus("Attached execution canceled and closed.")
					return a, a.refreshCmd()
				}
				a.setStatus("Execution exited. Press ctrl+g q to leave attach mode.")
			}
			a.attach.exited = true
			return a, a.refreshCmd()
		case *agentv1.AttachEvent_Error:
			a.setError(errors.New(payload.Error.GetMessage()))
			a.detach()
			return a, nil
		case *agentv1.AttachEvent_Ack:
			return a, a.recvAttachCmd()
		default:
			return a, a.recvAttachCmd()
		}
	case tea.KeyMsg:
		if a.attach != nil {
			return a.handleAttachKey(msg)
		}
		if a.focus == focusDetail && a.detailInputOpen {
			if key.Matches(msg, a.keys.NextFocus) {
				a.focus = nextFocus(a.section, a.focus)
				a.syncFormFocus()
				return a, nil
			}
			if key.Matches(msg, a.keys.PrevFocus) {
				a.focus = prevFocus(a.section, a.focus)
				a.syncFormFocus()
				return a, nil
			}
			return a.handleDetailKey(msg)
		}
		deletePressed := key.Matches(msg, a.keys.Delete)
		clearPressed := key.Matches(msg, a.keys.ClearAll)
		if a.pendingAction != destructiveActionNone && !deletePressed && !clearPressed {
			a.clearPendingDestructive()
		}
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}
		if key.Matches(msg, a.keys.ToggleHelp) {
			a.showHelp = !a.showHelp
			return a, nil
		}
		if key.Matches(msg, a.keys.Refresh) && a.focus != focusForm {
			a.loading = true
			return a, a.refreshCmd()
		}
		if key.Matches(msg, a.keys.NextFocus) {
			if a.focus == focusForm {
				if maxField := a.activeFormFieldCount() - 1; maxField > 0 && a.formCursor < maxField {
					a.formCursor++
					a.syncFormFocus()
					return a, nil
				}
			}
			a.focus = nextFocus(a.section, a.focus)
			a.syncFormFocus()
			return a, nil
		}
		if key.Matches(msg, a.keys.PrevFocus) {
			if a.focus == focusForm {
				if a.formCursor > 0 {
					a.formCursor--
					a.syncFormFocus()
					return a, nil
				}
			}
			a.focus = prevFocus(a.section, a.focus)
			a.syncFormFocus()
			return a, nil
		}
		if key.Matches(msg, a.keys.Back) {
			a.err = nil
			return a, nil
		}
		if key.Matches(msg, a.keys.RunningOnly) && a.section == sectionExecutions && a.focus == focusList {
			a.runningOnly = !a.runningOnly
			return a, a.refreshCmd()
		}
		if key.Matches(msg, a.keys.ViewOutput) && a.section == sectionExecutions && a.focus == focusDetail {
			a.showOutput = !a.showOutput
			return a, a.loadDetailForSelection()
		}
		if deletePressed && (a.section == sectionExecutions || a.section == sectionTransfers) && (a.focus == focusList || a.focus == focusDetail) {
			selected := a.selectedItem()
			if selected == nil {
				return a, nil
			}
			if a.pendingAction == destructiveActionDelete && a.pendingTargetID == selected.GetExecutionId() {
				a.loading = true
				a.clearPendingDestructive()
				return a, deleteHistoryCmd(a.client, selected.GetExecutionId())
			}
			a.pendingAction = destructiveActionDelete
			a.pendingTargetID = selected.GetExecutionId()
			a.setStatus(fmt.Sprintf("Press x again to delete %s from history.", selected.GetExecutionId()))
			return a, nil
		}
		if clearPressed && (a.section == sectionExecutions || a.section == sectionTransfers) {
			if a.pendingAction == destructiveActionClearHistory {
				a.loading = true
				a.clearPendingDestructive()
				return a, clearHistoryCmd(a.client)
			}
			a.pendingAction = destructiveActionClearHistory
			a.pendingTargetID = ""
			a.setStatus("Press X again to clear finished history for the authenticated identity. Running items are preserved.")
			return a, nil
		}
		if key.Matches(msg, a.keys.Cancel) && (a.section == sectionExecutions || a.section == sectionTransfers) && a.focus == focusList {
			selected := a.selectedItem()
			if selected == nil || selected.GetState() != agentv1.ExecutionState_EXECUTION_STATE_RUNNING {
				a.setError(errors.New("selected item is not running"))
				return a, nil
			}
			a.loading = true
			return a, cancelExecutionCmd(a.client, selected.GetExecutionId(), 5*time.Second)
		}
		if key.Matches(msg, a.keys.Attach) && a.section == sectionExecutions && a.focus == focusList {
			selected := a.selectedItem()
			if selected == nil {
				return a, nil
			}
			if selected.GetState() != agentv1.ExecutionState_EXECUTION_STATE_RUNNING {
				a.setError(errors.New("selected execution is not running"))
				return a, nil
			}
			a.loading = true
			return a, attachConnectCmd(a.client, selected.GetExecutionId())
		}
		switch a.focus {
		case focusSidebar:
			return a.handleSidebarKey(msg)
		case focusList:
			switch a.section {
			case sectionExecutions, sectionTransfers:
				return a.handleListKey(msg)
			case sectionNewCommand:
				return a.handleCommandForm(msg)
			case sectionNewTransfer:
				return a.handleTransferForm(msg)
			case sectionConnection:
				return a.handleConnectionForm(msg)
			}
		case focusForm:
			switch a.section {
			case sectionNewCommand:
				return a.handleCommandForm(msg)
			case sectionNewTransfer:
				return a.handleTransferForm(msg)
			case sectionConnection:
				return a.handleConnectionForm(msg)
			}
		case focusDetail:
			return a.handleDetailKey(msg)
		}
	case executionCanceledMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.setStatus(fmt.Sprintf("Cancel requested for %s", msg.exec.GetExecutionId()))
		return a, a.refreshCmd()
	case historyDeletedMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		if a.selectedID[a.section] == msg.executionID {
			a.selectedID[a.section] = ""
		}
		a.lastSubmittedID = ""
		a.setStatus(fmt.Sprintf("Deleted %s from history.", msg.executionID))
		return a, a.refreshCmd()
	case historyClearedMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		a.selectedID[sectionExecutions] = ""
		a.selectedID[sectionTransfers] = ""
		a.lastSubmittedID = ""
		a.setStatus(fmt.Sprintf("Cleared %d history item(s); skipped %d running item(s).", msg.deletedCount, msg.skippedRunningCount))
		return a, a.refreshCmd()
	case stdinWrittenMsg:
		a.loading = false
		if msg.err != nil {
			a.setError(msg.err)
			return a, nil
		}
		if msg.eof && msg.bytesSent > 0 {
			a.setStatus(fmt.Sprintf("Sent %d stdin byte(s) and EOF to %s.", msg.bytesSent, msg.executionID))
		} else if msg.eof {
			a.setStatus(fmt.Sprintf("Sent EOF to %s.", msg.executionID))
		} else {
			a.setStatus(fmt.Sprintf("Sent %d stdin byte(s) to %s.", msg.bytesSent, msg.executionID))
		}
		return a, a.refreshCmd()
	}
	return a, nil
}

type executionCanceledMsg struct {
	exec *agentv1.Execution
	err  error
}

func (a *app) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sections := []section{sectionExecutions, sectionTransfers, sectionNewCommand, sectionNewTransfer, sectionConnection}
	current := 0
	for idx, sec := range sections {
		if sec == a.section {
			current = idx
			break
		}
	}
	switch {
	case key.Matches(msg, a.keys.Up):
		if current > 0 {
			a.closeDetailInput()
			a.section = sections[current-1]
			a.formCursor = 0
			a.syncFormFocus()
			return a, a.loadDetailForSelection()
		}
	case key.Matches(msg, a.keys.Down):
		if current < len(sections)-1 {
			a.closeDetailInput()
			a.section = sections[current+1]
			a.formCursor = 0
			a.syncFormFocus()
			return a, a.loadDetailForSelection()
		}
	}
	return a, nil
}

func (a *app) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := a.currentList()
	if len(items) == 0 {
		return a, nil
	}
	idx := a.selection[a.section]
	switch {
	case key.Matches(msg, a.keys.Up):
		if idx > 0 {
			a.closeDetailInput()
			a.selection[a.section] = idx - 1
			a.lastSubmittedID = ""
			a.selectedID[a.section] = primarySelectionID(items[a.selection[a.section]])
			return a, a.loadDetailForSelection()
		}
	case key.Matches(msg, a.keys.Down):
		if idx < len(items)-1 {
			a.closeDetailInput()
			a.selection[a.section] = idx + 1
			a.lastSubmittedID = ""
			a.selectedID[a.section] = primarySelectionID(items[a.selection[a.section]])
			return a, a.loadDetailForSelection()
		}
	}
	return a, nil
}

func (a *app) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.detailInputOpen {
		switch {
		case key.Matches(msg, a.keys.Back):
			a.closeDetailInput()
			a.setStatus("Stdin input canceled.")
			return a, nil
		case key.Matches(msg, a.keys.Submit):
			selected := a.selectedExecutionForInput()
			if selected == nil {
				a.closeDetailInput()
				a.setError(errors.New("selected item does not accept stdin"))
				return a, nil
			}
			payload := []byte(a.detailInput.Value() + "\n")
			a.closeDetailInput()
			a.loading = true
			return a, writeStdinCmd(a.client, selected.GetExecutionId(), payload, false)
		default:
			var cmd tea.Cmd
			a.detailInput, cmd = a.detailInput.Update(msg)
			return a, cmd
		}
	}
	if key.Matches(msg, a.keys.WriteStdin) {
		selected := a.selectedExecutionForInput()
		if selected == nil {
			a.setError(errors.New("selected item does not accept stdin"))
			return a, nil
		}
		a.detailInputOpen = true
		a.detailInput.SetValue("")
		a.detailInput.Focus()
		a.setStatus(fmt.Sprintf("Entering stdin for %s. Press enter to send one line, esc to cancel.", selected.GetExecutionId()))
		return a, nil
	}
	if key.Matches(msg, a.keys.SendEOF) {
		selected := a.selectedExecutionForInput()
		if selected == nil {
			a.setError(errors.New("selected item does not accept stdin"))
			return a, nil
		}
		a.loading = true
		return a, writeStdinCmd(a.client, selected.GetExecutionId(), nil, true)
	}
	var cmd tea.Cmd
	a.detailViewport, cmd = a.detailViewport.Update(msg)
	return a, cmd
}

func (a *app) handleCommandForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, a.keys.NextMode) {
		a.commandMode = commandMode((int(a.commandMode) + 1) % 3)
		a.formCursor = 0
		a.syncFormFocus()
		return a, nil
	}
	if key.Matches(msg, a.keys.PrevMode) {
		a.commandMode = commandMode((int(a.commandMode) + 2) % 3)
		a.formCursor = 0
		a.syncFormFocus()
		return a, nil
	}
	if key.Matches(msg, a.keys.Submit) {
		cmd, err := a.submitCommandForm()
		if err != nil {
			a.setError(err)
			return a, nil
		}
		a.resetCommandForm()
		a.loading = true
		return a, cmd
	}
	for i := range a.commandInputs {
		var cmd tea.Cmd
		a.commandInputs[i], cmd = a.commandInputs[i].Update(msg)
		if cmd != nil {
			return a, cmd
		}
	}
	return a, nil
}

func (a *app) handleTransferForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, a.keys.NextMode) {
		a.transferMode = transferMode((int(a.transferMode) + 1) % 3)
		a.formCursor = 0
		a.syncFormFocus()
		return a, nil
	}
	if key.Matches(msg, a.keys.PrevMode) {
		a.transferMode = transferMode((int(a.transferMode) + 2) % 3)
		a.formCursor = 0
		a.syncFormFocus()
		return a, nil
	}
	if key.Matches(msg, a.keys.Submit) {
		cmd, err := a.submitTransferForm()
		if err != nil {
			a.setError(err)
			return a, nil
		}
		a.resetTransferForm()
		a.loading = true
		return a, cmd
	}
	for i := range a.transferInputs {
		var cmd tea.Cmd
		a.transferInputs[i], cmd = a.transferInputs[i].Update(msg)
		if cmd != nil {
			return a, cmd
		}
	}
	return a, nil
}

func (a *app) handleConnectionForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	for i := range a.connectInputs {
		var cmd tea.Cmd
		a.connectInputs[i], cmd = a.connectInputs[i].Update(msg)
		if cmd != nil {
			return a, cmd
		}
	}
	return a, nil
}

func (a *app) handleAttachKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.attach == nil {
		return a, nil
	}
	if a.attach.awaitingCmd {
		a.attach.awaitingCmd = false
		switch msg.String() {
		case "q":
			a.detach()
			a.setStatus("Detached from live session.")
			return a, nil
		case "c":
			if err := a.attach.session.CancelExecution(); err != nil {
				a.setError(err)
			} else {
				a.attach.closeOnExit = true
				a.setStatus("Cancel request sent to attached execution. Waiting for exit…")
			}
			return a, nil
		case "h":
			a.setStatus("Attach mode: keys go to the remote process. Use ctrl+g q to detach, ctrl+g c to cancel, ctrl+g h for help, ctrl+d to send EOF.")
			return a, nil
		default:
			a.setStatus("Attach mode command canceled.")
			return a, nil
		}
	}

	if msg.Type == tea.KeyCtrlG {
		a.attach.awaitingCmd = true
		a.setStatus("Attach prefix received. q detach, c cancel, h help.")
		return a, nil
	}
	if msg.Type == tea.KeyCtrlC {
		a.detach()
		return a, nil
	}
	if msg.Type == tea.KeyCtrlD {
		return a, sendAttachInputCmd(a.attach.session, nil, true)
	}

	if !a.attach.usesPTY && (msg.Type == tea.KeyUp || msg.Type == tea.KeyDown || msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown) {
		var cmd tea.Cmd
		a.attach.viewport, cmd = a.attach.viewport.Update(msg)
		return a, cmd
	}

	data := keyMsgBytes(msg)
	if len(data) == 0 {
		return a, nil
	}
	return a, sendAttachInputCmd(a.attach.session, data, false)
}

func (a *app) View() string {
	if !a.ready {
		return "Loading…"
	}
	if a.attach != nil {
		return a.renderAttachView()
	}

	sidebarWidth := min(22, max(18, a.width/6))
	contentWidth := max(20, a.width-sidebarWidth-7)
	topHeight := max(8, (a.height-8)/2)
	bottomHeight := max(8, a.height-topHeight-8)

	sidebar := a.renderSidebar(sidebarWidth)
	main := a.renderMainPane(contentWidth, topHeight)
	detail := a.renderDetailPane(contentWidth, bottomHeight)
	footer := a.renderFooter(contentWidth + sidebarWidth + 3)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, lipgloss.NewStyle().Width(1).Render(" "), lipgloss.JoinVertical(lipgloss.Left, main, detail))
	return a.styles.doc.Render(lipgloss.JoinVertical(lipgloss.Left, body, footer))
}

func (a *app) renderSidebar(width int) string {
	items := []struct {
		sec   section
		label string
	}{
		{sectionExecutions, "Executions"},
		{sectionTransfers, "Transfers"},
		{sectionNewCommand, "New Command"},
		{sectionNewTransfer, "New Transfer"},
		{sectionConnection, "Connection"},
	}
	lines := []string{a.panelHeading("Navigation", focusSidebar), a.styles.subtitle.Render("tab cycles panels; j/k drives the focused panel"), ""}
	for _, item := range items {
		style := a.styles.navItem
		if item.sec == a.section {
			style = a.styles.navActive
		}
		lines = append(lines, style.Width(width-4).Render(item.label))
	}
	lines = append(lines, "", a.styles.subtitle.Render("Status"), renderKeyValue(a.styles, "Address", a.conn.Address), renderKeyValue(a.styles, "Selection", a.selectionLabel()))
	return a.panelStyle(focusSidebar).Width(width).Height(max(8, a.height-6)).Render(strings.Join(lines, "\n"))
}

func (a *app) renderMainPane(width, height int) string {
	switch a.section {
	case sectionExecutions, sectionTransfers:
		return a.panelStyle(focusList).Width(width).Height(height).Render(a.panelHeading("Main", focusList) + "\n" + a.renderListView(width-4, height-3))
	case sectionNewCommand:
		return a.panelStyle(focusForm).Width(width).Height(height).Render(a.panelHeading("Main", focusForm) + "\n" + a.renderCommandForm(width-4, height-3))
	case sectionNewTransfer:
		return a.panelStyle(focusForm).Width(width).Height(height).Render(a.panelHeading("Main", focusForm) + "\n" + a.renderTransferForm(width-4, height-3))
	case sectionConnection:
		return a.panelStyle(focusForm).Width(width).Height(height).Render(a.panelHeading("Main", focusForm) + "\n" + a.renderConnectionView(width-4, height-3))
	default:
		return a.panelStyle(focusList).Width(width).Height(height).Render("")
	}
}

func (a *app) renderDetailPane(width, height int) string {
	title := "Execution Detail"
	if a.section == sectionTransfers {
		title = "Transfer Detail"
	} else if a.section == sectionNewCommand || a.section == sectionNewTransfer || a.section == sectionConnection {
		title = "Detail"
	}
	content := a.detailViewport.View()
	if a.loading {
		content = a.spinner.View() + " Loading…\n\n" + content
	}
	if a.detailInputOpen {
		content += "\n\n" + a.styles.panelTitle.Render("Send stdin line") + "\n" + a.detailInput.View() + "\n" + a.styles.muted.Render("Enter sends one line terminated with newline. Esc cancels. Use 'e' outside the prompt to send EOF only.")
	}
	return a.panelStyle(focusDetail).Width(width).Height(height).Render(a.panelHeading(title, focusDetail) + "\n" + content)
}

func (a *app) renderFooter(width int) string {
	statusStyle := a.styles.success
	if a.err != nil {
		statusStyle = a.styles.error
	}
	statusText := a.status
	if a.err != nil {
		statusText = a.err.Error()
	}
	if statusText == "" {
		statusText = "Ready"
	}
	short := []key.Binding{a.keys.NextFocus, a.keys.PrevFocus, a.keys.Up, a.keys.Down, a.keys.Refresh, a.keys.ToggleHelp, a.keys.Quit}
	if a.focus == focusList {
		short = []key.Binding{a.keys.NextFocus, a.keys.PrevFocus, a.keys.Up, a.keys.Down, a.keys.Attach, a.keys.Cancel, a.keys.Delete, a.keys.ClearAll}
	}
	if a.focus == focusDetail {
		short = []key.Binding{a.keys.NextFocus, a.keys.PrevFocus, a.keys.Up, a.keys.Down, a.keys.ViewOutput, a.keys.WriteStdin, a.keys.SendEOF, a.keys.Delete}
	}
	if a.focus == focusForm {
		short = []key.Binding{a.keys.NextFocus, a.keys.PrevFocus, a.keys.Submit, a.keys.NextMode, a.keys.PrevMode, a.keys.ToggleHelp, a.keys.Quit}
	}
	helpText := a.help.ShortHelpView(short)
	if a.showHelp {
		helpText = a.help.FullHelpView([][]key.Binding{{a.keys.Up, a.keys.Down, a.keys.NextFocus, a.keys.PrevFocus}, {a.keys.Refresh, a.keys.Attach, a.keys.Cancel, a.keys.Delete, a.keys.ClearAll}, {a.keys.ViewOutput, a.keys.WriteStdin, a.keys.SendEOF, a.keys.RunningOnly, a.keys.Submit, a.keys.NextMode, a.keys.PrevMode}, {a.keys.ToggleHelp, a.keys.Quit}})
	}
	lines := []string{statusStyle.Render(statusText), helpText}
	return a.styles.footer.Width(width).Render(strings.Join(lines, "\n"))
}

func (a *app) renderListView(width, height int) string {
	title := "Executions"
	items := a.executions
	if a.section == sectionTransfers {
		title = "Transfers"
		items = a.transfers
	}
	lines := []string{a.styles.panelTitle.Render(title), ""}
	if len(items) == 0 {
		lines = append(lines, a.styles.muted.Render("No items match the current view."))
		return strings.Join(lines, "\n")
	}
	lines = append(lines, renderListHeader(a.section))
	selected := a.selection[a.section]
	start := 0
	if selected > height-7 {
		start = selected - (height - 7)
	}
	end := min(len(items), start+max(1, height-5))
	for i := start; i < end; i++ {
		row := renderExecutionRow(items[i])
		if i == selected {
			row = a.styles.selectedRow.Width(width).Render(row)
		}
		lines = append(lines, row)
	}
	if a.lastSubmittedID != "" {
		lines = append(lines, "", a.styles.muted.Render("Last submitted: "+a.lastSubmittedID))
	}
	return strings.Join(lines, "\n")
}

func (a *app) renderCommandForm(width, height int) string {
	modes := renderModeTabs(
		a.styles,
		modeTab{label: "argv", active: a.commandMode == commandModeArgv},
		modeTab{label: "shell", active: a.commandMode == commandModeShell},
		modeTab{label: "session", active: a.commandMode == commandModeSession},
	)
	lines := []string{a.styles.panelTitle.Render("New Command"), modes, ""}
	switch a.commandMode {
	case commandModeArgv:
		lines = append(lines,
			"Binary:", a.commandInputs[0].View(),
			"Args:", a.commandInputs[1].View(),
			"", a.styles.muted.Render("Submit with enter. Args are split with shell-style quoting."),
		)
	case commandModeShell:
		lines = append(lines,
			"Shell Binary:", a.commandInputs[0].View(),
			"Command:", a.commandInputs[1].View(),
			"Use PTY:", a.commandInputs[3].View(),
			"", a.styles.muted.Render("Shell binary can be blank if the daemon side provides a default. PTY merges terminal-style output into one stream."),
		)
	case commandModeSession:
		lines = append(lines,
			"Shell Binary:", a.commandInputs[0].View(),
			"Shell Args:", a.commandInputs[2].View(),
			"Use PTY:", a.commandInputs[3].View(),
			"", a.styles.muted.Render("Start a persistent shell session, then attach with 'a' from Executions. PTY enables prompt-oriented terminal behavior."),
		)
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (a *app) renderTransferForm(width, height int) string {
	modes := renderModeTabs(
		a.styles,
		modeTab{label: "upload", active: a.transferMode == transferModeUpload},
		modeTab{label: "download", active: a.transferMode == transferModeDownload},
		modeTab{label: "archive", active: a.transferMode == transferModeArchive},
	)
	lines := []string{a.styles.panelTitle.Render("New Transfer"), modes, ""}
	switch a.transferMode {
	case transferModeUpload:
		lines = append(lines,
			"Local Path:", a.transferInputs[0].View(),
			"Remote Path:", a.transferInputs[1].View(),
			"", a.styles.muted.Render("Submit with enter. Uploads overwrite by default, matching cmdractl."),
		)
	case transferModeDownload:
		lines = append(lines,
			"Local Path:", a.transferInputs[0].View(),
			"Remote Path:", a.transferInputs[1].View(),
			"Chunk Size:", a.transferInputs[2].View(),
		)
	case transferModeArchive:
		lines = append(lines,
			"Local Archive Path:", a.transferInputs[0].View(),
			"Remote Paths (comma-separated):", a.transferInputs[1].View(),
			"Chunk Size:", a.transferInputs[2].View(),
		)
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (a *app) renderConnectionView(width, height int) string {
	lines := []string{a.styles.panelTitle.Render("Connection"), "", "Address:", a.connectInputs[0].View(), "CA PEM:", a.connectInputs[1].View(), "Client Cert PEM:", a.connectInputs[2].View(), "Client Key PEM:", a.connectInputs[3].View(), "Server Name Override:", a.connectInputs[4].View(), "", a.styles.muted.Render("Connection editing is informational in v1. Launch cmdraui with the desired TLS flags."), "", renderKeyValue(a.styles, "TLS verify", fmt.Sprintf("skip=%t", a.conn.InsecureSkipVerify))}
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

func (a *app) renderAttachView() string {
	if a.attach == nil {
		return ""
	}
	banner := a.styles.attachBanner.Width(max(10, a.width-4)).Render(fmt.Sprintf("Attached to %s  |  ctrl+g q detach  ctrl+g c cancel  ctrl+g h help  ctrl+d EOF", a.attach.executionID))
	view := a.attach.viewport.View()
	if a.attach.usesPTY && a.attach.terminal != nil {
		view = a.attach.terminal.View()
		if a.debugPTY {
			stats := a.attach.terminal.Stats()
			debugLine := a.styles.muted.Render(fmt.Sprintf("writes=%d bytes=%d last_write=%s last_bytes=%d frames=%d last_render=%s", stats.Writes, stats.Bytes, stats.LastWrite.Round(time.Millisecond), stats.LastWriteBytes, stats.Frames, stats.LastRender.Round(time.Millisecond)))
			view = lipgloss.JoinVertical(lipgloss.Left, view, "", debugLine)
		}
	}
	return a.styles.doc.Render(lipgloss.JoinVertical(lipgloss.Left, banner, view))
}

func (a *app) resize() {
	if !a.ready {
		return
	}
	sidebarWidth := min(22, max(18, a.width/6))
	contentWidth := max(20, a.width-sidebarWidth-7)
	topHeight := max(8, (a.height-8)/2)
	bottomHeight := max(8, a.height-topHeight-8)
	a.detailViewport.Width = max(1, contentWidth-4)
	a.detailViewport.Height = max(1, bottomHeight-4)
	if a.attach != nil {
		a.attach.viewport.Width = max(1, a.width-4)
		a.attach.viewport.Height = max(1, a.height-4)
		if a.attach.usesPTY && a.attach.terminal != nil {
			a.attach.terminal.Resize(max(1, a.width-4), max(1, a.height-4))
		}
	}
	_ = topHeight
}

func (a *app) refreshCmd() tea.Cmd {
	a.loading = true
	return loadExecutionsCmd(a.client, a.runningOnly)
}

func loadExecutionsCmd(client *cmdraclient.Client, runningOnly bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var filter *bool
		filter = &runningOnly
		items, err := client.ListExecutions(ctx, filter)
		return loadExecutionsMsg{items: items, err: err}
	}
}

func (a *app) loadDetailForSelection() tea.Cmd {
	selected := a.selectedItem()
	if selected == nil {
		a.detailMeta = nil
		a.detailOutput = nil
		a.syncDetailViewport()
		return nil
	}
	return loadDetailCmd(a.client, selected.GetExecutionId(), a.section == sectionExecutions && a.showOutput)
}

func loadDetailCmd(client *cmdraclient.Client, executionID string, includeOutput bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		execMeta, err := client.GetExecution(ctx, executionID)
		if err != nil {
			return loadDetailMsg{err: err}
		}
		msg := loadDetailMsg{exec: execMeta}
		if includeOutput {
			chunks, err := client.ReadOutput(ctx, executionID, 0, false)
			if err != nil {
				return loadDetailMsg{err: err}
			}
			msg.output = renderOutputChunks(chunks)
		}
		return msg
	}
}

func writeStdinCmd(client *cmdraclient.Client, executionID string, data []byte, eof bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.WriteStdin(ctx, executionID, data, eof)
		return stdinWrittenMsg{executionID: executionID, bytesSent: len(data), eof: eof, err: err}
	}
}

func (a *app) submitCommandForm() (tea.Cmd, error) {
	switch a.commandMode {
	case commandModeArgv:
		binary := strings.TrimSpace(a.commandInputs[0].Value())
		if binary == "" {
			return nil, errors.New("binary is required")
		}
		args, err := parseWords(a.commandInputs[1].Value())
		if err != nil {
			return nil, err
		}
		return startArgvCmd(a.client, binary, args), nil
	case commandModeShell:
		command := strings.TrimSpace(a.commandInputs[1].Value())
		if command == "" {
			return nil, errors.New("command is required")
		}
		usePTY, err := parseBoolInput(a.commandInputs[3].Value())
		if err != nil {
			return nil, err
		}
		rows, cols := a.commandPTYSize()
		return startShellCmd(a.client, strings.TrimSpace(a.commandInputs[0].Value()), command, usePTY, rows, cols), nil
	case commandModeSession:
		shell := strings.TrimSpace(a.commandInputs[0].Value())
		if shell == "" {
			return nil, errors.New("shell binary is required")
		}
		args, err := parseWords(a.commandInputs[2].Value())
		if err != nil {
			return nil, err
		}
		usePTY, err := parseBoolInput(a.commandInputs[3].Value())
		if err != nil {
			return nil, err
		}
		rows, cols := a.commandPTYSize()
		return startSessionCmd(a.client, shell, args, usePTY, rows, cols), nil
	default:
		return nil, errors.New("unknown command mode")
	}
}

func (a *app) submitTransferForm() (tea.Cmd, error) {
	localPath := strings.TrimSpace(a.transferInputs[0].Value())
	remoteSpec := strings.TrimSpace(a.transferInputs[1].Value())
	if localPath == "" {
		return nil, errors.New("local path is required")
	}
	chunkSize := 32 * 1024
	if raw := strings.TrimSpace(a.transferInputs[2].Value()); raw != "" {
		var err error
		_, err = fmt.Sscanf(raw, "%d", &chunkSize)
		if err != nil {
			return nil, errors.New("chunk size must be an integer")
		}
	}
	switch a.transferMode {
	case transferModeUpload:
		if remoteSpec == "" {
			return nil, errors.New("remote path is required")
		}
		return uploadFileCmd(a.client, localPath, remoteSpec), nil
	case transferModeDownload:
		if remoteSpec == "" {
			return nil, errors.New("remote path is required")
		}
		return downloadFileCmd(a.client, remoteSpec, localPath, chunkSize), nil
	case transferModeArchive:
		if remoteSpec == "" {
			return nil, errors.New("at least one remote path is required")
		}
		parts := splitCSV(remoteSpec)
		if len(parts) == 0 {
			return nil, errors.New("at least one remote path is required")
		}
		return downloadArchiveCmd(a.client, parts, localPath, chunkSize), nil
	default:
		return nil, errors.New("unknown transfer mode")
	}
}

func startArgvCmd(client *cmdraclient.Client, binary string, args []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		execMeta, err := client.StartArgv(ctx, binary, args)
		return startExecutionMsg{exec: execMeta, err: err}
	}
}

func startShellCmd(client *cmdraclient.Client, shellBinary, command string, usePTY bool, ptyRows, ptyCols uint32) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		execMeta, err := client.StartShellCommandWithOptions(ctx, shellBinary, command, cmdraclient.ShellOptions{
			UsePTY:  usePTY,
			PTYRows: ptyRows,
			PTYCols: ptyCols,
		})
		return startExecutionMsg{exec: execMeta, err: err}
	}
}

func startSessionCmd(client *cmdraclient.Client, shellBinary string, args []string, usePTY bool, ptyRows, ptyCols uint32) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		execMeta, err := client.StartShellSessionWithOptions(ctx, shellBinary, args, cmdraclient.ShellOptions{
			UsePTY:  usePTY,
			PTYRows: ptyRows,
			PTYCols: ptyCols,
		})
		return startExecutionMsg{exec: execMeta, err: err}
	}
}

func uploadFileCmd(client *cmdraclient.Client, localPath, remotePath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		overwrite := true
		resp, err := client.UploadFile(ctx, localPath, remotePath, cmdraclient.UploadOptions{Overwrite: &overwrite})
		return uploadDoneMsg{resp: resp, err: err}
	}
}

func downloadFileCmd(client *cmdraclient.Client, remotePath, localPath string, chunkSize int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := client.DownloadFile(ctx, remotePath, localPath, cmdraclient.DownloadOptions{ChunkSize: chunkSize})
		return downloadDoneMsg{localPath: localPath, remote: []string{remotePath}, resp: resp, err: err}
	}
}

func downloadArchiveCmd(client *cmdraclient.Client, paths []string, localPath string, chunkSize int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := client.DownloadArchive(ctx, paths, localPath, cmdraclient.DownloadOptions{ChunkSize: chunkSize})
		return downloadDoneMsg{localPath: localPath, remote: paths, resp: resp, err: err}
	}
}

func cancelExecutionCmd(client *cmdraclient.Client, executionID string, grace time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		execMeta, err := client.CancelExecution(ctx, executionID, grace)
		return executionCanceledMsg{exec: execMeta, err: err}
	}
}

func deleteHistoryCmd(client *cmdraclient.Client, executionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DeleteExecution(ctx, executionID)
		return historyDeletedMsg{executionID: executionID, err: err}
	}
}

func clearHistoryCmd(client *cmdraclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := client.ClearHistory(ctx)
		if err != nil {
			return historyClearedMsg{err: err}
		}
		return historyClearedMsg{
			deletedCount:        result.DeletedCount,
			skippedRunningCount: result.SkippedRunningCount,
		}
	}
}

func attachConnectCmd(client *cmdraclient.Client, executionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		session, err := client.Attach(ctx, executionID, true, 0)
		if err != nil {
			cancel()
			return attachConnectedMsg{err: err}
		}
		evt, err := session.Recv()
		if err != nil {
			cancel()
			return attachConnectedMsg{err: err}
		}
		ack, ok := evt.GetPayload().(*agentv1.AttachEvent_Ack)
		if !ok {
			cancel()
			return attachConnectedMsg{err: errors.New("attach did not return an ack")}
		}
		return attachConnectedMsg{session: session, cancel: cancel, ack: ack.Ack.GetExecution()}
	}
}

func (a *app) recvAttachCmd() tea.Cmd {
	if a.attach == nil {
		return nil
	}
	session := a.attach.session
	return func() tea.Msg {
		evt, err := session.Recv()
		return attachEventMsg{event: evt, err: err}
	}
}

func sendAttachInputCmd(session attachSession, data []byte, eof bool) tea.Cmd {
	return func() tea.Msg {
		if err := session.SendStdin(data, eof); err != nil {
			return attachEventMsg{err: err}
		}
		return nil
	}
}

func attachResizeCmd(session attachSession, rows, cols uint32) tea.Cmd {
	return func() tea.Msg {
		if rows == 0 || cols == 0 {
			return nil
		}
		if err := session.ResizePTY(rows, cols); err != nil {
			return attachEventMsg{err: err}
		}
		return nil
	}
}

func (a *app) syncDetailViewport() {
	if a.detailViewport.Width == 0 || a.detailViewport.Height == 0 {
		return
	}
	if a.detailInputOpen && a.selectedExecutionForInput() == nil {
		a.closeDetailInput()
	}
	if a.detailMeta == nil {
		a.detailKey = ""
		switch a.section {
		case sectionNewCommand:
			a.detailViewport.SetContent("Use tab to focus the main form. Then use tab and shift+tab to move between fields, [ and ] to switch command mode, and enter to submit.")
		case sectionNewTransfer:
			a.detailViewport.SetContent("Use tab to focus the main form. Then use tab and shift+tab to move between fields, [ and ] to switch transfer mode, and enter to submit.")
		case sectionConnection:
			a.detailViewport.SetContent("This panel shows the active connection settings. In v1 it is informational; restart cmdraui with different flags to reconnect.")
		default:
			a.detailViewport.SetContent("Select an item to inspect it.")
		}
		return
	}
	previousKey := a.detailKey
	nextKey := a.detailMeta.GetExecutionId()
	if nextKey == "" {
		nextKey = fmt.Sprintf("%v", a.section)
	}
	previousYOffset := a.detailViewport.YOffset
	lines := []string{renderExecutionSummary(a.detailMeta)}
	if len(a.detailOutput) > 0 {
		lines = append(lines, "", "Output:", strings.Join(a.detailOutput, "\n"))
	}
	if selected := a.selectedExecutionForInput(); selected != nil {
		lines = append(lines, "", "Detail Actions:", "  i  send one stdin line", "  e  send EOF")
	}
	a.detailKey = nextKey
	a.detailViewport.SetContent(strings.Join(lines, "\n"))
	if previousKey != nextKey {
		a.detailViewport.GotoTop()
	} else {
		maxOffset := max(0, a.detailViewport.TotalLineCount()-a.detailViewport.Height)
		if previousYOffset > maxOffset {
			previousYOffset = maxOffset
		}
		a.detailViewport.SetYOffset(previousYOffset)
	}
}

func (a *app) commandPTYSize() (uint32, uint32) {
	return uint32(max(1, a.height-8)), uint32(max(1, a.width-10))
}

func (a *app) attachPTYSize() (uint32, uint32) {
	return uint32(max(1, a.height-4)), uint32(max(1, a.width-4))
}

func (a *app) selectedExecutionForInput() *agentv1.Execution {
	if a.section != sectionExecutions {
		return nil
	}
	selected := a.selectedItem()
	if selected == nil {
		return nil
	}
	switch selected.GetKind() {
	case agentv1.ExecutionKind_EXECUTION_KIND_COMMAND:
		if strings.TrimSpace(selected.GetCommandShell()) == "" {
			return nil
		}
	case agentv1.ExecutionKind_EXECUTION_KIND_SHELL_SESSION:
	default:
		return nil
	}
	if selected.GetState() != agentv1.ExecutionState_EXECUTION_STATE_RUNNING {
		return nil
	}
	return selected
}

func (a *app) selectedItem() *agentv1.Execution {
	items := a.currentList()
	if len(items) == 0 {
		return nil
	}
	idx := a.selection[a.section]
	if idx < 0 || idx >= len(items) {
		return nil
	}
	return items[idx]
}

func (a *app) currentList() []*agentv1.Execution {
	switch a.section {
	case sectionExecutions:
		return a.executions
	case sectionTransfers:
		return a.transfers
	default:
		return nil
	}
}

func (a *app) selectionLabel() string {
	if item := a.selectedItem(); item != nil {
		return item.GetExecutionId()
	}
	return "none"
}

func selectedFromList(items []*agentv1.Execution, idx int) *agentv1.Execution {
	if idx < 0 || idx >= len(items) {
		return nil
	}
	return items[idx]
}

func (a *app) setStatus(status string) {
	a.status = status
	a.err = nil
	a.statusWhen = time.Now()
}

func (a *app) setError(err error) {
	a.err = err
	a.statusWhen = time.Now()
}

func (a *app) clearPendingDestructive() {
	a.pendingAction = destructiveActionNone
	a.pendingTargetID = ""
}

func (a *app) clampSelections() {
	usedSubmittedID := false
	for _, sec := range []section{sectionExecutions, sectionTransfers} {
		items := a.executions
		if sec == sectionTransfers {
			items = a.transfers
		}
		if len(items) == 0 {
			a.selection[sec] = 0
			a.selectedID[sec] = ""
			continue
		}
		if a.lastSubmittedID != "" {
			for idx, item := range items {
				if matchesSelectionID(item, a.lastSubmittedID) {
					a.selection[sec] = idx
					a.selectedID[sec] = primarySelectionID(item)
					usedSubmittedID = true
					goto nextSection
				}
			}
		}
		if targetID := a.selectedID[sec]; targetID != "" {
			for idx, item := range items {
				if matchesSelectionID(item, targetID) {
					a.selection[sec] = idx
					a.selectedID[sec] = primarySelectionID(item)
					break
				}
			}
		}
		if a.selection[sec] >= len(items) {
			a.selection[sec] = len(items) - 1
		}
		if a.selection[sec] < 0 {
			a.selection[sec] = 0
		}
		a.selectedID[sec] = primarySelectionID(items[a.selection[sec]])
	nextSection:
	}
	if usedSubmittedID {
		a.lastSubmittedID = ""
	}
}

func primarySelectionID(item *agentv1.Execution) string {
	if item == nil {
		return ""
	}
	if item.GetExecutionId() != "" {
		return item.GetExecutionId()
	}
	if item.GetLastUploadTransferId() != "" {
		return item.GetLastUploadTransferId()
	}
	if item.GetLastDownloadTransferId() != "" {
		return item.GetLastDownloadTransferId()
	}
	return ""
}

func matchesSelectionID(item *agentv1.Execution, id string) bool {
	if item == nil || id == "" {
		return false
	}
	return item.GetExecutionId() == id || item.GetLastUploadTransferId() == id || item.GetLastDownloadTransferId() == id
}

func (a *app) syncFormFocus() {
	focusField := func(fields []textinput.Model, active []int, cursor int, focused bool) []textinput.Model {
		target := -1
		if len(active) > 0 {
			cursor = min(cursor, len(active)-1)
			target = active[cursor]
		}
		for i := range fields {
			if i == target && focused {
				fields[i].Focus()
			} else {
				fields[i].Blur()
			}
		}
		return fields
	}
	commandActive := []int{0, 1}
	switch a.commandMode {
	case commandModeShell:
		commandActive = []int{0, 1, 3}
	case commandModeSession:
		commandActive = []int{0, 2, 3}
	}
	transferActive := []int{0, 1}
	if a.transferMode != transferModeUpload {
		transferActive = []int{0, 1, 2}
	}
	connectionActive := []int{0, 1, 2, 3, 4}
	a.commandInputs = focusField(a.commandInputs, commandActive, a.formCursor, a.focus == focusForm && a.section == sectionNewCommand)
	a.transferInputs = focusField(a.transferInputs, transferActive, a.formCursor, a.focus == focusForm && a.section == sectionNewTransfer)
	a.connectInputs = focusField(a.connectInputs, connectionActive, a.formCursor, a.focus == focusForm && a.section == sectionConnection)
}

func (a *app) activeFormFieldCount() int {
	switch a.section {
	case sectionNewCommand:
		switch a.commandMode {
		case commandModeSession:
			return 3
		case commandModeShell:
			return 3
		default:
			return 2
		}
	case sectionNewTransfer:
		switch a.transferMode {
		case transferModeUpload:
			return 2
		default:
			return 3
		}
	case sectionConnection:
		return len(a.connectInputs)
	default:
		return 0
	}
}

func (a *app) resetCommandForm() {
	for i := range a.commandInputs {
		a.commandInputs[i].SetValue("")
	}
	a.formCursor = 0
	a.syncFormFocus()
}

func (a *app) resetTransferForm() {
	for i := range a.transferInputs {
		a.transferInputs[i].SetValue("")
	}
	a.formCursor = 0
	a.syncFormFocus()
}

func (a *app) closeDetailInput() {
	a.detailInputOpen = false
	a.detailInput.SetValue("")
	a.detailInput.Blur()
}

func (a *app) detach() {
	if a.attach == nil {
		return
	}
	_ = a.attach.session.CloseSend()
	a.attach.cancel()
	a.attach = nil
}

func splitExecutions(items []*agentv1.Execution) (executions []*agentv1.Execution, transfers []*agentv1.Execution) {
	for _, item := range items {
		switch item.GetKind() {
		case agentv1.ExecutionKind_EXECUTION_KIND_UPLOAD,
			agentv1.ExecutionKind_EXECUTION_KIND_DOWNLOAD,
			agentv1.ExecutionKind_EXECUTION_KIND_ARCHIVE_DOWNLOAD:
			transfers = append(transfers, item)
		default:
			executions = append(executions, item)
		}
	}
	sortExecutions(executions)
	sortExecutions(transfers)
	return executions, transfers
}

func sortExecutions(items []*agentv1.Execution) {
	sort.SliceStable(items, func(i, j int) bool {
		ti := items[i].GetStartedAt().AsTime()
		tj := items[j].GetStartedAt().AsTime()
		return ti.After(tj)
	})
}

func renderOutputChunks(chunks []*agentv1.OutputChunk) []string {
	lines := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.GetEof() {
			continue
		}
		src := strings.TrimPrefix(chunk.GetSource().String(), "OUTPUT_SOURCE_")
		for _, line := range strings.Split(strings.ReplaceAll(string(chunk.GetData()), "\r\n", "\n"), "\n") {
			if line == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("[%s] %s", src, line))
		}
	}
	return lines
}

func renderExecutionSummary(execMeta *agentv1.Execution) string {
	if execMeta == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("ID: %s", execMeta.GetExecutionId()),
		fmt.Sprintf("Kind: %s", strings.TrimPrefix(execMeta.GetKind().String(), "EXECUTION_KIND_")),
		fmt.Sprintf("State: %s", strings.TrimPrefix(execMeta.GetState().String(), "EXECUTION_STATE_")),
		fmt.Sprintf("Owner CN: %s", execMeta.GetOwnerCn()),
	}
	if execMeta.GetPid() != 0 {
		lines = append(lines, fmt.Sprintf("PID: %d", execMeta.GetPid()))
	}
	if execMeta.GetStartedAt() != nil {
		lines = append(lines, fmt.Sprintf("Started: %s", execMeta.GetStartedAt().AsTime().Format(time.RFC3339)))
	}
	if execMeta.GetEndedAt() != nil {
		lines = append(lines, fmt.Sprintf("Ended: %s", execMeta.GetEndedAt().AsTime().Format(time.RFC3339)))
	}
	if len(execMeta.GetCommandArgv()) > 0 {
		lines = append(lines, fmt.Sprintf("Argv: %s", strings.Join(execMeta.GetCommandArgv(), " ")))
	}
	if execMeta.GetCommandShell() != "" {
		lines = append(lines, fmt.Sprintf("Shell: %s", execMeta.GetCommandShell()))
	}
	if execMeta.GetUsesPty() {
		lines = append(lines, "Uses PTY: true")
		if execMeta.GetPtyRows() > 0 && execMeta.GetPtyCols() > 0 {
			lines = append(lines, fmt.Sprintf("PTY Size: %dx%d", execMeta.GetPtyRows(), execMeta.GetPtyCols()))
		}
	}
	if execMeta.GetLastUploadTransferId() != "" {
		lines = append(lines, fmt.Sprintf("Upload Transfer ID: %s", execMeta.GetLastUploadTransferId()))
	}
	if execMeta.GetLastUploadLocalPath() != "" || execMeta.GetLastUploadRemotePath() != "" {
		lines = append(lines, fmt.Sprintf("Upload Paths: %s -> %s", execMeta.GetLastUploadLocalPath(), execMeta.GetLastUploadRemotePath()))
	}
	if execMeta.GetLastDownloadTransferId() != "" {
		lines = append(lines, fmt.Sprintf("Download Transfer ID: %s", execMeta.GetLastDownloadTransferId()))
	}
	if execMeta.GetLastDownloadRemotePath() != "" || execMeta.GetLastDownloadLocalPath() != "" {
		lines = append(lines, fmt.Sprintf("Download Paths: %s -> %s", execMeta.GetLastDownloadRemotePath(), execMeta.GetLastDownloadLocalPath()))
	}
	if execMeta.GetTransferDirection() != "" {
		lines = append(lines, fmt.Sprintf("Transfer: %s %d/%d bytes", execMeta.GetTransferDirection(), execMeta.GetTransferProgressBytes(), execMeta.GetTransferTotalBytes()))
	}
	lines = append(lines, fmt.Sprintf("Output Size: %d", execMeta.GetOutputSizeBytes()), fmt.Sprintf("Exit Code: %d", execMeta.GetExitCode()))
	if execMeta.GetSignal() != "" {
		lines = append(lines, fmt.Sprintf("Signal: %s", execMeta.GetSignal()))
	}
	if execMeta.GetErrorMessage() != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", execMeta.GetErrorMessage()))
	}
	return strings.Join(lines, "\n")
}

func renderExecutionRow(execMeta *agentv1.Execution) string {
	name := commandLabel(execMeta)
	return fmt.Sprintf("%-24s %-9s %-18s %s", trimRight(execMeta.GetExecutionId(), 24), trimRight(strings.TrimPrefix(execMeta.GetState().String(), "EXECUTION_STATE_"), 9), trimRight(strings.TrimPrefix(execMeta.GetKind().String(), "EXECUTION_KIND_"), 18), trimRight(name, 80))
}

func renderListHeader(sec section) string {
	if sec == sectionTransfers {
		return fmt.Sprintf("%-24s %-9s %-18s %s", "ID", "STATE", "KIND", "PATHS")
	}
	return fmt.Sprintf("%-24s %-9s %-18s %s", "ID", "STATE", "KIND", "COMMAND")
}

func commandLabel(execMeta *agentv1.Execution) string {
	switch execMeta.GetKind() {
	case agentv1.ExecutionKind_EXECUTION_KIND_UPLOAD:
		return fmt.Sprintf("%s -> %s", execMeta.GetLastUploadLocalPath(), execMeta.GetLastUploadRemotePath())
	case agentv1.ExecutionKind_EXECUTION_KIND_DOWNLOAD, agentv1.ExecutionKind_EXECUTION_KIND_ARCHIVE_DOWNLOAD:
		return fmt.Sprintf("%s -> %s", execMeta.GetLastDownloadRemotePath(), execMeta.GetLastDownloadLocalPath())
	default:
		if len(execMeta.GetCommandArgv()) > 0 {
			return strings.Join(execMeta.GetCommandArgv(), " ")
		}
		return execMeta.GetCommandShell()
	}
}

func parseBoolInput(raw string) (bool, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "", "false", "f", "no", "n", "0":
		return false, nil
	case "true", "t", "yes", "y", "1":
		return true, nil
	default:
		return false, errors.New("PTY must be true or false")
	}
}

func renderKeyValue(s styles, keyName, value string) string {
	return s.status.Render(keyName+":") + " " + s.statusValue.Render(value)
}

func renderChip(s styles, label string, active bool) string {
	if active {
		return s.chipActive.Render(label)
	}
	return s.chip.Render(label)
}

type modeTab struct {
	label  string
	active bool
}

func renderModeTabs(s styles, tabs ...modeTab) string {
	rendered := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if tab.active {
			rendered = append(rendered, s.chipActive.Render(tab.label))
			continue
		}
		rendered = append(rendered, s.chip.Render(tab.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
}

func parseWords(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var parts []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range raw {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, errors.New("unclosed quote in arguments")
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func keyMsgBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyEnter:
		return []byte("\n")
	case tea.KeyTab:
		return []byte("\t")
	case tea.KeyShiftTab:
		return []byte("\x1b[Z")
	case tea.KeySpace:
		return []byte(" ")
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyEsc:
		return []byte{0x1b}
	case tea.KeyCtrlA:
		return []byte{0x01}
	case tea.KeyCtrlB:
		return []byte{0x02}
	case tea.KeyCtrlE:
		return []byte{0x05}
	case tea.KeyCtrlF:
		return []byte{0x06}
	case tea.KeyCtrlK:
		return []byte{0x0b}
	case tea.KeyCtrlL:
		return []byte{0x0c}
	case tea.KeyCtrlN:
		return []byte{0x0e}
	case tea.KeyCtrlP:
		return []byte{0x10}
	case tea.KeyCtrlU:
		return []byte{0x15}
	case tea.KeyCtrlW:
		return []byte{0x17}
	case tea.KeyCtrlY:
		return []byte{0x19}
	case tea.KeyF1:
		return []byte("\x1bOP")
	case tea.KeyF2:
		return []byte("\x1bOQ")
	case tea.KeyF3:
		return []byte("\x1bOR")
	case tea.KeyF4:
		return []byte("\x1bOS")
	case tea.KeyF5:
		return []byte("\x1b[15~")
	case tea.KeyF6:
		return []byte("\x1b[17~")
	case tea.KeyF7:
		return []byte("\x1b[18~")
	case tea.KeyF8:
		return []byte("\x1b[19~")
	case tea.KeyF9:
		return []byte("\x1b[20~")
	case tea.KeyF10:
		return []byte("\x1b[21~")
	case tea.KeyF11:
		return []byte("\x1b[23~")
	case tea.KeyF12:
		return []byte("\x1b[24~")
	case tea.KeyRunes:
		if msg.Paste {
			return append(append([]byte("\x1b[200~"), []byte(string(msg.Runes))...), []byte("\x1b[201~")...)
		}
		return []byte(string(msg.Runes))
	default:
		switch msg.String() {
		case "up":
			return []byte("\x1b[A")
		case "down":
			return []byte("\x1b[B")
		case "left":
			return []byte("\x1b[D")
		case "right":
			return []byte("\x1b[C")
		case "ctrl+up":
			return []byte("\x1b[1;5A")
		case "ctrl+down":
			return []byte("\x1b[1;5B")
		case "ctrl+right":
			return []byte("\x1b[1;5C")
		case "ctrl+left":
			return []byte("\x1b[1;5D")
		case "ctrl+home":
			return []byte("\x1b[1;5H")
		case "ctrl+end":
			return []byte("\x1b[1;5F")
		case "ctrl+pgup":
			return []byte("\x1b[5;5~")
		case "ctrl+pgdown":
			return []byte("\x1b[6;5~")
		}
	}
	return nil
}

func literalMsgCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func nextFocus(s section, current focusArea) focusArea {
	switch current {
	case focusSidebar:
		if s == sectionExecutions || s == sectionTransfers {
			return focusList
		}
		return focusForm
	case focusList, focusForm:
		return focusDetail
	default:
		return focusSidebar
	}
}

func (a *app) panelHeading(title string, panel focusArea) string {
	return a.styles.panelTitle.Render(title)
}

func (a *app) panelStyle(panel focusArea) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if a.focus == panel {
		borderColor = lipgloss.Color("63")
	}
	return a.styles.border.Copy().BorderForeground(borderColor)
}

func prevFocus(s section, current focusArea) focusArea {
	switch current {
	case focusSidebar:
		return focusDetail
	case focusDetail:
		if s == sectionExecutions || s == sectionTransfers {
			return focusList
		}
		return focusForm
	default:
		return focusSidebar
	}
}

func trimRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Run launches the cmdra terminal UI.
func Run(client *cmdraclient.Client, cfg cmdraclient.DialConfig) error {
	p := tea.NewProgram(New(client, cfg), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}
