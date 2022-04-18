package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var cursorBg = lipgloss.AdaptiveColor{Light: "#dddddd", Dark: "#444444"}
var markerStyle = lipgloss.NewStyle().Width(2)
var hashStyle = lipgloss.NewStyle().
	Width(9).
	PaddingRight(1).
	Foreground(lipgloss.Color("#dd77dd"))
var ageStyle = lipgloss.NewStyle().
	Align(lipgloss.Right).
	Width(4).
	PaddingRight(1)
var nameStyle = lipgloss.NewStyle().
	Width(21).
	PaddingRight(1)
var subjectStyle = lipgloss.NewStyle().Inline(true)
var statusStyle = lipgloss.NewStyle().
	Inline(true).
	Background(lipgloss.AdaptiveColor{Light: "#ddddff", Dark: "#444466"})
var statStyle = lipgloss.NewStyle().Inline(true)

type view string

const (
	commitsView view = "commits"
	statsView        = "stats"
	diffView         = "diff"
)

type listModel struct {
	count  int
	first  int
	last   int
	height int
	cursor int
	marked int
}

type appModel struct {
	height  int
	width   int
	history []view

	commits     []commit
	commitsList listModel

	stats     []stat
	statsList listModel

	diff      []string
	diffModel listModel

	status string
}

func (m appModel) renderCommit(index int) string {
	c := m.commits[index]

	ctime := time.Unix(c.Timestamp, 0)
	var age string
	years, months, days, hours, mins, secs, _ := Elapsed(ctime, time.Now())
	if years > 0 {
		age = fmt.Sprintf("%dY", years)
	} else if months > 0 {
		age = fmt.Sprintf("%dM", months)
	} else if days > 0 {
		age = fmt.Sprintf("%dD", days)
	} else if hours > 0 {
		age = fmt.Sprintf("%dh", hours)
	} else if mins > 0 {
		age = fmt.Sprintf("%dm", mins)
	} else {
		age = fmt.Sprintf("%ds", secs)
	}

	name := c.AuthorName
	if len(name) > 20 {
		parts := strings.Split(name, " ")
		if len(parts) >= 3 {
			name = fmt.Sprintf("%s ", parts[0])
			for i := 0; i < len(parts)-2; i++ {
				name += fmt.Sprintf("%c", parts[i][0])
			}
			name += fmt.Sprintf(" %s", parts[len(parts)-1])
		} else if len(parts) == 2 {
			name = fmt.Sprintf("%c %s", parts[0][0], parts[1])
		} else {
			name = name[0:20]
		}
	}

	marker := ""
	if index == m.commitsList.marked {
		marker = "*"
	}

	subjectStyle.Width(m.width -
		markerStyle.GetWidth() -
		hashStyle.GetWidth() -
		ageStyle.GetWidth() -
		nameStyle.GetWidth())

	if index == m.commitsList.cursor {
		markerStyle.Background(cursorBg)
		hashStyle.Background(cursorBg)
		ageStyle.Background(cursorBg)
		nameStyle.Background(cursorBg)
		subjectStyle.Background(cursorBg)
	} else {
		markerStyle.UnsetBackground()
		hashStyle.UnsetBackground()
		ageStyle.UnsetBackground()
		nameStyle.UnsetBackground()
		subjectStyle.UnsetBackground()
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		markerStyle.Render(marker),
		hashStyle.Render(c.Commit[0:8]),
		ageStyle.Render(age),
		nameStyle.Render(name),
		subjectStyle.Render(c.Subject),
	)
}

func (m appModel) renderStat(index int) string {
	s := m.stats[index]

	statStyle.Width(m.width)

	if index == m.statsList.cursor {
		statStyle.Background(cursorBg)
		markerStyle.Background(cursorBg)
	} else {
		statStyle.UnsetBackground()
		markerStyle.UnsetBackground()
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		markerStyle.Render(" "),
		statStyle.Render(s.Path),
	)
}

func (m appModel) currentView() view {
	return m.history[len(m.history)-1]
}

func (m appModel) pushView(view view) appModel {
	m.history = append(m.history, view)
	return m
}

func (m appModel) popView() appModel {
	m.history = m.history[:len(m.history)-1]
	return m
}

func (m listModel) setHeight(height int) listModel {
	if height < m.count {
		m.height = height
	} else {
		m.height = m.count
	}
	m.last = m.first + m.height - 1
	if m.cursor > m.last-1 {
		m.last = m.cursor + 1
		m.first = m.last - m.height + 1
	}
	return m
}

func (m listModel) nextPage() listModel {
	if m.cursor != -1 {
		m.cursor += m.height
	}

	m.first += m.height
	m.last += m.height
	if m.last >= m.count {
		m.last = m.count - 1
		m.first = m.last - m.height + 1
	}

	if m.cursor != -1 {
		if m.cursor > m.last {
			m.cursor = m.last
		}
	}

	return m
}

func (m listModel) prevPage() listModel {
	if m.cursor != -1 {
		m.cursor -= m.height
	}

	m.first -= m.height
	m.last -= m.height
	if m.first < 0 {
		m.first = 0
		m.last = m.first + m.height - 1
	}

	if m.cursor != -1 {
		if m.cursor < 0 {
			m.cursor = 0
		}
	}

	return m
}

func (m listModel) nextItem() listModel {
	if m.cursor == -1 {
		// Not using cursor
		if m.last < m.count-1 {
			m.first += 1
			m.last += 1
		}
	} else if m.cursor < m.count-1 {
		m.cursor += 1
		if m.cursor > m.last-1 {
			m.first += 1
			m.last += 1
		}
	}
	return m
}

func (m listModel) prevItem() listModel {
	if m.cursor == -1 {
		// Not using cursor
		if m.first > 0 {
			m.first -= 1
			m.last -= 1
		}
	} else if m.cursor > 0 {
		m.cursor -= 1
		if m.cursor < m.first {
			m.first -= 1
			m.last -= 1
		}
	}
	return m
}

func (m appModel) Init() tea.Cmd {
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		statusStyle.Width(m.width)

		if m.currentView() == commitsView {
			m.commitsList = m.commitsList.setHeight(m.height)
		} else if m.currentView() == statsView {
			m.statsList = m.statsList.setHeight(m.height)
		} else if m.currentView() == diffView {
			m.diffModel = m.diffModel.setHeight(m.height)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case " ":
			if m.currentView() == commitsView {
				if m.commitsList.marked == m.commitsList.cursor {
					m.commitsList.marked = -1
				} else {
					m.commitsList.marked = m.commitsList.cursor
				}
			}

		case "ctrl+f":
			if m.currentView() == commitsView {
				m.commitsList = m.commitsList.nextPage()
			} else if m.currentView() == statsView {
				m.statsList = m.statsList.nextPage()
			} else if m.currentView() == diffView {
				m.diffModel = m.diffModel.nextPage()
			}

		case "ctrl+u":
			if m.currentView() == commitsView {
				m.commitsList = m.commitsList.prevPage()
			} else if m.currentView() == statsView {
				m.statsList = m.statsList.prevPage()
			} else if m.currentView() == diffView {
				m.diffModel = m.diffModel.prevPage()
			}

		case "j":
			if m.currentView() == commitsView {
				m.commitsList = m.commitsList.nextItem()
			} else if m.currentView() == statsView {
				m.statsList = m.statsList.nextItem()
			} else if m.currentView() == diffView {
				m.diffModel = m.diffModel.nextItem()
			}

		case "k":
			if m.currentView() == commitsView {
				m.commitsList = m.commitsList.prevItem()
			} else if m.currentView() == statsView {
				m.statsList = m.statsList.prevItem()
			} else if m.currentView() == diffView {
				m.diffModel = m.diffModel.prevItem()
			}

		case "enter":
			if m.currentView() == commitsView {
				commitA := m.commits[m.commitsList.cursor].Commit[:8]
				commitB := "HEAD"
				m.stats = gitDiffStat(commitA, commitB)
				m = m.pushView(statsView)
				m.statsList = listModel{
					count:  len(m.stats),
					marked: -1}
				m.statsList = m.statsList.setHeight(m.height)
				m.status = fmt.Sprintf("%s..%s", commitA, commitB)
			} else if m.currentView() == statsView {
				commitA := m.commits[m.commitsList.cursor].Commit[:8]
				commitB := "HEAD"
				path := m.stats[m.statsList.cursor].Path

				m.diff = gitDiff(commitA, commitB, path)
				m.diffModel = listModel{
					count:  len(m.diff),
					marked: -1,
					cursor: -1}
				m.diffModel = m.diffModel.setHeight(m.height)
				m.status = fmt.Sprintf(
					"%s..%s: %s",
					commitA,
					commitB,
					path)
				m = m.pushView(diffView)
			}

		case "esc":
			if len(m.history) == 1 {
				return m, tea.Quit
			}
			m = m.popView()
		}
	}

	return m, nil
}

func (m appModel) View() string {
	mainSection := ""

	switch m.currentView() {
	case commitsView:
		var lines []string
		for i := m.commitsList.first; i < m.commitsList.last; i++ {
			lines = append(lines, m.renderCommit(i))
		}
		mainSection = lipgloss.JoinVertical(lipgloss.Left, lines...)

	case statsView:
		var lines []string
		for i := m.statsList.first; i < m.statsList.last; i++ {
			lines = append(lines, m.renderStat(i))
		}
		mainSection = lipgloss.JoinVertical(lipgloss.Left, lines...)

	case diffView:
		var lines []string
		for i := m.diffModel.first; i < m.diffModel.last; i++ {
			lines = append(lines, m.diff[i])
		}
		mainSection = lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.PlaceVertical(m.height-1, lipgloss.Top, mainSection),
		statusStyle.Render(m.status),
	)
}

func main() {
	commits := gitLog()

	m := appModel{
		history: []view{commitsView},
		commits: commits,
		commitsList: listModel{
			count:  len(commits),
			marked: -1,
		},
		statsList: listModel{
			marked: -1,
		},
		status: "",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
