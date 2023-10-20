package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

type Style struct {
	green    func(strs ...string) string
	yellow   func(strs ...string) string
	red      func(strs ...string) string
	gradient progress.Option
}

func ParseStyle(style string) (*Style, error) {
	colors := strings.Split(style, ",")
	if len(colors) != 3 {
		return nil, fmt.Errorf("three colors must be provided for the style, found %d (%q)", len(colors), style)
	}
	s := &Style{}
	s.green = lipgloss.NewStyle().Foreground(lipgloss.Color(colors[0])).Render
	s.yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(colors[1])).Render
	s.red = lipgloss.NewStyle().Foreground(lipgloss.Color(colors[2])).Render

	s.gradient = progress.WithGradient(colors[2], colors[0])
	return s, nil
}
