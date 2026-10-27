package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"lopy/internal/agent"
)

type modelChoices struct {
	generation int
	models     []agent.ModelChoice
	errs       map[string]error
}

func (m *model) openPicker(kind string) tea.Cmd {
	m.pasteGeneration++
	m.pasting = false
	m.menuKind, m.selected = kind, 0
	m.menu = nil
	m.input.Reset()
	m.menuGeneration++
	if kind == "login" {
		for name := range m.a.Settings().Providers {
			m.menu = append(m.menu, command{name: name, description: "Provider"})
		}
		sort.Slice(m.menu, func(i, j int) bool { return m.menu[i].name < m.menu[j].name })
		m.menu = append(m.menu, command{name: "Add", icon: "+", description: "Create provider"})
		return nil
	}
	m.effort = 0
	for i, e := range agent.Efforts {
		if e == m.a.Settings().Effort {
			m.effort = i
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	m.menuCancel = cancel
	generation, snapshot := m.menuGeneration, *m.a
	return func() tea.Msg {
		defer cancel()
		models, errs := snapshot.AllModels(ctx)
		return modelChoices{generation, models, errs}
	}
}
func (m *model) closePicker() {
	m.pasteGeneration++
	m.pasting = false
	if m.menuCancel != nil {
		m.menuCancel()
		m.menuCancel = nil
	}
	m.menuKind, m.menu = "", nil
	m.setup = nil
	m.input.Placeholder = "Ask anything, or type / for commands"
	m.input.Reset()
}
func (m model) pickerKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.menuKind == "" {
		return m, nil, false
	}
	switch key.String() {
	case "esc", "ctrl+c":
		m.closePicker()
	case "left", "right":
		if m.menuKind != "model" {
			return m, nil, false
		}
		delta := 1
		if key.String() == "left" {
			delta = -1
		}
		m.effort = (m.effort + delta + len(agent.Efforts)) % len(agent.Efforts)
	case "up", "down", "tab":
		matches := m.completions()
		if len(matches) > 0 {
			delta := 1
			if key.String() == "up" {
				delta = -1
			}
			m.selected = (m.selected + delta + len(matches)) % len(matches)
		}
	case "enter":
		if m.pasting {
			return m, nil, true
		}
		if m.menuKind == "setup" {
			value := strings.TrimSpace(m.input.Value())
			if len(m.setup) == 2 { // API type is chosen from the list
				matches := m.completions()
				if len(matches) == 0 {
					return m, nil, true
				}
				value = matches[m.selected%len(matches)].name
			} else if value == "" && len(m.setup) != 3 {
				return m, nil, true
			}
			m.setup = append(m.setup, value)
			m.input.Reset()
			m.selected = 0
			m.menu = nil
			prompts := []string{"Provider name", "HTTP(S) base URL", "API type: choose one (Up/Down, Enter)", "API key (optional)"}
			if len(m.setup) < 4 {
				m.input.Placeholder = prompts[len(m.setup)]
				if len(m.setup) == 2 {
					m.menu = []command{
						{name: "responses", description: "OpenAI Responses API"},
						{name: "chat_completions", description: "OpenAI-compatible Chat Completions API"},
					}
				}
				break
			}
			err := m.a.AddProvider(m.setup[0], agent.Provider{BaseURL: m.setup[1], APIType: m.setup[2], APIKey: m.setup[3]})
			m.closePicker()
			if err != nil {
				m.add("error", err.Error())
			} else {
				m.add("notice", "Provider created. Choose a model with /model.")
			}
			break
		}
		matches := m.completions()
		if len(matches) == 0 {
			return m, nil, true
		}
		choice, kind := matches[m.selected%len(matches)].name, m.menuKind
		if kind == "login" && matches[m.selected%len(matches)].icon == "+" {
			m.menuKind = "setup"
			m.menu = nil
			m.selected = 0
			m.input.Reset()
			m.input.Placeholder = "Provider name"
			break
		}
		settings := m.a.Settings()
		if kind == "model" {
			settings.Provider = matches[m.selected%len(matches)].description
			settings.Model, settings.Effort = choice, agent.Efforts[m.effort]
		} else {
			settings.Provider = choice
		}
		err := m.a.Select(settings.Provider, settings.Model, settings.Effort)
		m.closePicker()
		if err != nil {
			m.add("error", err.Error())
		} else {
			m.add("notice", fmt.Sprintf("Selected %s · %s/%s", settings.Provider, settings.Model, settings.Effort))
		}
	default:
		return m, nil, false
	}
	m.refresh(true)
	return m, nil, true
}
