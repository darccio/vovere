package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type statusTUIMsg string

type errTUIMsg struct {
	err error
}

func (e errTUIMsg) Error() string {
	return e.err.Error()
}

type tuiModel struct {
	handler func() (string, error)
	spinner spinner.Model
	status  string
	err     error
}

func newTUIModel() tuiModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return tuiModel{
		spinner: s,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		// TODO: add Lipgloss styling
		// TODO: refactor code to reduce coupling
		func() tea.Msg {
			msg, err := m.handler()
			if err != nil {
				return errTUIMsg{
					err: err,
				}
			}
			return statusTUIMsg(msg)
		},
	)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errTUIMsg:
		m.err = msg.err
		return m, tea.Quit
	case statusTUIMsg:
		m.status = string(msg)
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m tuiModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("error: %v\n", m.err)
	}
	if m.status != "" {
		return fmt.Sprintf("%s\n", m.status)
	}
	return fmt.Sprintf("%s Working...\n", m.spinner.View())
}
