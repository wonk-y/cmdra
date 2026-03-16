package tui

import (
	"io"
	"testing"

	agentv1 "cmdra/gen/agent/v1"
	"cmdra/pkg/cmdraclient"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeAttachSession struct {
	cancelCalls int
	closeCalls  int
	stdinCalls  int
	resizeCalls int
}

func (f *fakeAttachSession) Recv() (*agentv1.AttachEvent, error) {
	return nil, io.EOF
}

func (f *fakeAttachSession) SendStdin(_ []byte, _ bool) error {
	f.stdinCalls++
	return nil
}

func (f *fakeAttachSession) CancelExecution() error {
	f.cancelCalls++
	return nil
}

func (f *fakeAttachSession) ResizePTY(_, _ uint32) error {
	f.resizeCalls++
	return nil
}

func (f *fakeAttachSession) CloseSend() error {
	f.closeCalls++
	return nil
}

func TestAttachCancelClosesOnExit(t *testing.T) {
	model := New(nil, cmdraclient.DialConfig{}).(*app)
	session := &fakeAttachSession{}
	cancelled := false
	model.attach = &attachState{
		executionID: "exec-1",
		usesPTY:     true,
		session:     session,
		cancel:      func() { cancelled = true },
	}

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if session.cancelCalls != 1 {
		t.Fatalf("expected one cancel call, got %d", session.cancelCalls)
	}
	if model.attach == nil || !model.attach.closeOnExit {
		t.Fatalf("expected attach session to wait for close on exit")
	}

	exitEvt := &agentv1.AttachEvent{
		Payload: &agentv1.AttachEvent_Exit{
			Exit: &agentv1.ExitEvent{
				Execution: &agentv1.Execution{
					ExecutionId: "exec-1",
					State:       agentv1.ExecutionState_EXECUTION_STATE_CANCELED,
				},
			},
		},
	}
	_, _ = model.Update(attachEventMsg{event: exitEvt})

	if model.attach != nil {
		t.Fatal("expected attach session to close after canceled execution exited")
	}
	if session.closeCalls != 1 {
		t.Fatalf("expected CloseSend on detach, got %d", session.closeCalls)
	}
	if !cancelled {
		t.Fatal("expected attach context cancel to be invoked")
	}
}

func TestAttachRemoteExitKeepsViewUntilDetached(t *testing.T) {
	model := New(nil, cmdraclient.DialConfig{}).(*app)
	session := &fakeAttachSession{}
	model.attach = &attachState{
		executionID: "exec-2",
		usesPTY:     true,
		session:     session,
		cancel:      func() {},
	}

	exitEvt := &agentv1.AttachEvent{
		Payload: &agentv1.AttachEvent_Exit{
			Exit: &agentv1.ExitEvent{
				Execution: &agentv1.Execution{
					ExecutionId: "exec-2",
					State:       agentv1.ExecutionState_EXECUTION_STATE_EXITED,
				},
			},
		},
	}
	_, _ = model.Update(attachEventMsg{event: exitEvt})

	if model.attach == nil {
		t.Fatal("expected attach view to remain visible for non-cancel exit")
	}
	if !model.attach.exited {
		t.Fatal("expected attach session to be marked exited")
	}
}

func TestAttachDetachPrefixQClosesImmediately(t *testing.T) {
	model := New(nil, cmdraclient.DialConfig{}).(*app)
	session := &fakeAttachSession{}
	cancelled := false
	model.attach = &attachState{
		executionID: "exec-3",
		usesPTY:     false,
		session:     session,
		cancel:      func() { cancelled = true },
	}

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if model.attach != nil {
		t.Fatal("expected attach session to detach immediately")
	}
	if session.closeCalls != 1 {
		t.Fatalf("expected CloseSend on detach, got %d", session.closeCalls)
	}
	if !cancelled {
		t.Fatal("expected attach context cancel on detach")
	}
}

func TestKeyMsgBytesSupportsPasteAndExtendedKeys(t *testing.T) {
	paste := keyMsgBytes(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello"), Paste: true})
	if string(paste) != "\x1b[200~hello\x1b[201~" {
		t.Fatalf("unexpected bracketed paste sequence: %q", string(paste))
	}

	if got := string(keyMsgBytes(tea.KeyMsg{Type: tea.KeyF5})); got != "\x1b[15~" {
		t.Fatalf("unexpected F5 mapping: %q", got)
	}
	if got := string(keyMsgBytes(tea.KeyMsg{Type: tea.KeyCtrlU})); got != string([]byte{0x15}) {
		t.Fatalf("unexpected ctrl+u mapping: %q", got)
	}
	if got := string(keyMsgBytes(tea.KeyMsg{Type: tea.KeyCtrlRight})); got != "\x1b[1;5C" {
		t.Fatalf("unexpected ctrl+right mapping: %q", got)
	}
}

func TestCommandFormClearsFieldsAfterSuccessfulSubmit(t *testing.T) {
	model := New(nil, cmdraclient.DialConfig{}).(*app)
	model.section = sectionNewCommand
	model.focus = focusForm
	model.commandMode = commandModeShell
	model.formCursor = 1
	model.commandInputs[0].SetValue("/bin/sh")
	model.commandInputs[1].SetValue("printf 'hello\\n'")
	model.commandInputs[2].SetValue("--unused")
	model.commandInputs[3].SetValue("true")
	model.syncFormFocus()

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	for i, input := range model.commandInputs {
		if got := input.Value(); got != "" {
			t.Fatalf("expected command input %d to be cleared, got %q", i, got)
		}
	}
	if model.formCursor != 0 {
		t.Fatalf("expected form cursor reset to 0, got %d", model.formCursor)
	}
}

func TestTransferFormClearsFieldsAfterSuccessfulSubmit(t *testing.T) {
	model := New(nil, cmdraclient.DialConfig{}).(*app)
	model.section = sectionNewTransfer
	model.focus = focusForm
	model.transferMode = transferModeArchive
	model.formCursor = 2
	model.transferInputs[0].SetValue("./tmp.zip")
	model.transferInputs[1].SetValue("/tmp,/var/tmp")
	model.transferInputs[2].SetValue("65536")
	model.syncFormFocus()

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	for i, input := range model.transferInputs {
		if got := input.Value(); got != "" {
			t.Fatalf("expected transfer input %d to be cleared, got %q", i, got)
		}
	}
	if model.formCursor != 0 {
		t.Fatalf("expected form cursor reset to 0, got %d", model.formCursor)
	}
}
