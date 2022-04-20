package main

import (
	"fmt"
	"os"

	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view string

type watcherMessage struct {
	event string
	path  string
}

const (
	commitsView view = "commits"
	statsView        = "stats"
	diffView         = "diff"
)

type diffModel struct {
	path    string
	oldPath string
	list    listModel
}

type appModel struct {
	height       int
	width        int
	history      []view
	watcherReady bool

	commits     []commit
	commitsList listModel

	stats     []stat
	statsList listModel

	diff      []string
	diffModel diffModel

	status string
}

func trunc(s string, size int) string {
	if len(s) > size {
		return s[:size]
	}
	return s
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
		marker = "▶"
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
		branchStyle.Background(cursorBg)
		tagStyle.Background(cursorBg)
		refStyle.Background(cursorBg)
		subjectStyle.Background(cursorBg)
	} else {
		markerStyle.UnsetBackground()
		hashStyle.UnsetBackground()
		ageStyle.UnsetBackground()
		nameStyle.UnsetBackground()
		branchStyle.UnsetBackground()
		tagStyle.UnsetBackground()
		refStyle.UnsetBackground()
		subjectStyle.UnsetBackground()
	}

	branches := ""
	tags := ""
	refs := ""

	if c.Decoration != "" {
		info := parseDecoration(c.Decoration)
		for _, b := range info.branches {
			branches += fmt.Sprintf("[%s] ", b)
		}
		if branches != "" {
			branches = branchStyle.Render(branches)
		}

		for _, t := range info.tags {
			tags += fmt.Sprintf("<%s> ", t)
		}
		if tags != "" {
			tags = tagStyle.Render(tags)
		}

		for _, r := range info.refs {
			refs += fmt.Sprintf("{%s} ", r)
		}
		if refs != "" {
			refs = refStyle.Render(refs)
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		markerStyle.Render(marker),
		hashStyle.Render(c.Commit[0:8]),
		ageStyle.Render(age),
		nameStyle.Render(name),
		branches,
		tags,
		refs,
		subjectStyle.Render(c.Subject),
	)
}

func (m appModel) renderStat(index int) string {
	s := m.stats[index]

	statStyle.Width(m.width)

	statTypeStyle := statModStyle
	if (s.Change[0] == 'A') {
		statTypeStyle = statAddStyle
	} else if (s.Change[0] == 'D') {
		statTypeStyle = statRemStyle
	}

	if index == m.statsList.cursor {
		statStyle.Background(cursorBg)
		statTypeStyle.Background(cursorBg)
	} else {
		statStyle.UnsetBackground()
		statTypeStyle.UnsetBackground()
	}

	path := s.Path
	if s.OldPath != "" {
		path = path + " ← " + s.OldPath
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		statTypeStyle.Render(string(s.Change[0])),
		statStyle.Render(path),
	)
}

func (m appModel) renderDiffLine(index int) string {
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

func (m appModel) getCommitRange() (start, end string) {
	start = m.commits[m.commitsList.cursor].Commit
	end = ""

	if m.commitsList.marked >= 0 {
		if m.commitsList.marked > m.commitsList.cursor {
			end = start
			start = m.commits[m.commitsList.marked].Commit
		} else {
			end = m.commits[m.commitsList.marked].Commit
		}
	}

	return
}

func (m appModel) getStatus() string {
	switch m.currentView() {
	case commitsView:
		start, end := m.getCommitRange()
		if end == "" {
			end = "<index>"
		}
		return fmt.Sprintf("%s..%s", trunc(start, 8), trunc(end, 8))
	case statsView:
		start, end := m.getCommitRange()
		if end == "" {
			end = "<index>"
		}
		return fmt.Sprintf("%s..%s", trunc(start, 8), trunc(end, 8))
	case diffView:
		start, end := m.getCommitRange()
		if end == "" {
			end = "<index>"
		}
		path := m.stats[m.statsList.cursor].Path
		return fmt.Sprintf("%s..%s: %s", trunc(start, 8), trunc(end, 8), path)
	}

	return ""
}

func (m appModel) Init() tea.Cmd {
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.currentView() == commitsView {
			m.commitsList.setHeight(m.height)
		} else if m.currentView() == statsView {
			m.statsList.setHeight(m.height)
		} else if m.currentView() == diffView {
			m.diffModel.list.setHeight(m.height)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
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
				m.commitsList.nextPage()
			} else if m.currentView() == statsView {
				m.statsList.nextPage()
			} else if m.currentView() == diffView {
				m.diffModel.list.nextPage()
			}

		case "ctrl+u":
			if m.currentView() == commitsView {
				m.commitsList.prevPage()
			} else if m.currentView() == statsView {
				m.statsList.prevPage()
			} else if m.currentView() == diffView {
				m.diffModel.list.prevPage()
			}

		case "j", "down":
			if m.currentView() == commitsView {
				m.commitsList.nextItem()
			} else if m.currentView() == statsView {
				m.statsList.nextItem()
			} else if m.currentView() == diffView {
				m.diffModel.list.nextItem()
			}

		case "k", "up":
			if m.currentView() == commitsView {
				m.commitsList.prevItem()
			} else if m.currentView() == statsView {
				m.statsList.prevItem()
			} else if m.currentView() == diffView {
				m.diffModel.list.prevItem()
			}

		case "enter":
			if m.currentView() == commitsView {
				start, end := m.getCommitRange()
				m.stats = gitDiffStat(start, end)
				m.statsList = listModel{
					count:  len(m.stats),
					marked: -1}
				m.statsList.setHeight(m.height)
				m = m.pushView(statsView)
			} else if m.currentView() == statsView {
				start, end := m.getCommitRange()
				dm := m.stats[m.statsList.cursor]

				m.diff = gitDiff(start, end, dm.Path, dm.OldPath)

				m.diffModel = diffModel{
					path:    dm.Path,
					oldPath: dm.OldPath,
					list:    newCursorlessListModel(len(m.diff)),
				}
				m.diffModel.list.setHeight(m.height)
				m = m.pushView(diffView)
			}

		case "esc", "q":
			if len(m.history) == 1 {
				return m, tea.Quit
			}
			m = m.popView()
		}

	case watcherMessage:
		switch msg.event {
		case "ready":
			m.watcherReady = true
		case "filechange":
			if m.currentView() == diffView && m.diffModel.path == msg.path {
				start, end := m.getCommitRange()
				dm := m.diffModel
				m.diff = gitDiff(start, end, dm.path, dm.oldPath)
				m.diffModel.list.setCount(len(m.diff))
			}
		}
	}

	m.status = m.getStatus()

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
		for i := m.diffModel.list.first; i < m.diffModel.list.last; i++ {
			lines = append(lines, m.renderDiffLine(i))
		}
		mainSection = lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	statusRightStyle.Width(5)
	statusLeftStyle.Width(m.width - statusRightStyle.GetWidth())

	statusRight := "-"
	if m.watcherReady {
		statusRight = "#"
	}

	status := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusLeftStyle.Render(m.status),
		statusRightStyle.Render(statusRight),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.PlaceVertical(m.height-1, lipgloss.Top, mainSection),
		status,
	)
}

func main() {
	repoPath := "."
	if len(os.Args) > 1 {
		repoPath = os.Args[1]
	}
	os.Chdir(repoPath)

	commits := gitLog()

	m := appModel{
		history:     []view{commitsView},
		commits:     commits,
		commitsList: newListModel(len(commits)),
		statsList:   newListModel(0),
		status:      "",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	onNotify := func(event, path string) {
		if event == "ready" {
			p.Send(watcherMessage{event: "ready", path: ""})
		} else {
			p.Send(watcherMessage{event: "filechange", path: path})
		}
	}
	watcher := watchRepo(".", onNotify)
	defer watcher.Close()

	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
