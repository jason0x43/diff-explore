package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type diffModel struct {
	listModel
	diff        []string
	commits		commitRange
	path        string
	oldPath     string
	opts        diffOptions
}

func newDiffModel() diffModel {
	m := diffModel{}
	m.listModel.init(0, false)
	return m
}

func (m diffModel) name() string {
	return "diff"
}

func (m *diffModel) setDiff(c commitRange, s stat) {
	m.diff = gitDiff(c.start, c.end, s.Path, s.OldPath, m.opts)
	m.path = s.Path
	m.oldPath = s.OldPath
	m.listModel.init(len(m.diff), false)
}

func (m *diffModel) refresh() {
	m.diff = gitDiff(m.commits.start, m.commits.end, m.path, m.oldPath, m.opts)
	m.listModel.setCount(len(m.diff))
}

func (m diffModel) renderDiffLine(index int) string {
	d := m.diff[index]
	d = strings.ReplaceAll(d, "\t", "    ")

	if len(d) > 0 {
		switch d[0] {
		case '-':
			return diffRemStyle.Render(d)
		case '+':
			return diffAddStyle.Render(d)
		case '@':
			return diffSepStyle.Render(d)
		}
	}

	return diffNormalStyle.Render(d)
}

func (m diffModel) render() string {
	var lines []string
	for i := m.first; i < m.last; i++ {
		lines = append(lines, m.renderDiffLine(i))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}