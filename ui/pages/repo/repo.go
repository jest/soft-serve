package repo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/statusbar"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
)

type tab int

const (
	readmeTab tab = iota
	filesTab
	commitsTab
	branchesTab
	tagsTab
)

type UpdateStatusBarMsg struct{}

// RepoMsg is a message that contains a git.Repository.
type RepoMsg git.GitRepo

// RefMsg is a message that contains a git.Reference.
type RefMsg *ggit.Reference

// Repo is a view for a git repository.
type Repo struct {
	common       common.Common
	rs           git.GitRepoSource
	selectedRepo git.GitRepo
	selectedItem selection.Item
	activeTab    tab
	tabs         *tabs.Tabs
	statusbar    *statusbar.StatusBar
	readme       *code.Code
	log          *Log
	files        *Files
	ref          *ggit.Reference
}

// New returns a new Repo.
func New(common common.Common, rs git.GitRepoSource) *Repo {
	sb := statusbar.New(common)
	tb := tabs.New(common, []string{"Readme", "Files", "Commits", "Branches", "Tags"})
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	log := NewLog(common)
	files := NewFiles(common)
	r := &Repo{
		common:    common,
		rs:        rs,
		tabs:      tb,
		statusbar: sb,
		readme:    readme,
		log:       log,
		files:     files,
	}
	return r
}

// SetSize implements common.Component.
func (r *Repo) SetSize(width, height int) {
	r.common.SetSize(width, height)
	hm := r.common.Styles.RepoBody.GetVerticalFrameSize() +
		r.common.Styles.RepoHeader.GetHeight() +
		r.common.Styles.RepoHeader.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight() +
		r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	r.tabs.SetSize(width, height-hm)
	r.statusbar.SetSize(width, height-hm)
	r.readme.SetSize(width, height-hm)
	r.log.SetSize(width, height-hm)
	r.files.SetSize(width, height-hm)
}

// ShortHelp implements help.KeyMap.
func (r *Repo) ShortHelp() []key.Binding {
	b := make([]key.Binding, 0)
	tab := r.common.KeyMap.Section
	tab.SetHelp("tab", "switch tab")
	back := r.common.KeyMap.Back
	back.SetHelp("esc", "repos")
	b = append(b, back)
	b = append(b, tab)
	switch r.activeTab {
	case readmeTab:
		b = append(b, r.common.KeyMap.UpDown)
	case commitsTab:
		b = append(b, r.log.ShortHelp()...)
	}
	return b
}

// FullHelp implements help.KeyMap.
func (r *Repo) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	return b
}

// Init implements tea.View.
func (r *Repo) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (r *Repo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case selector.SelectMsg:
		switch msg.IdentifiableItem.(type) {
		case selection.Item:
			r.selectedItem = msg.IdentifiableItem.(selection.Item)
		}
	case RepoMsg:
		r.activeTab = 0
		r.selectedRepo = git.GitRepo(msg)
		r.readme.GotoTop()
		cmds = append(cmds,
			r.tabs.Init(),
			r.updateReadmeCmd,
			r.updateRefCmd,
			r.updateModels(msg),
		)
	case RefMsg:
		r.ref = msg
		cmds = append(cmds,
			r.updateStatusBarCmd,
			r.log.Init(),
			r.files.Init(),
			r.updateModels(msg),
		)
	case tabs.ActiveTabMsg:
		r.activeTab = tab(msg)
		if r.selectedRepo != nil {
			cmds = append(cmds, r.updateStatusBarCmd)
		}
	case tea.KeyMsg, tea.MouseMsg:
		if r.selectedRepo != nil {
			cmds = append(cmds, r.updateStatusBarCmd)
		}
	case FileItemsMsg:
		f, cmd := r.files.Update(msg)
		r.files = f.(*Files)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case LogCountMsg, LogItemsMsg:
		l, cmd := r.log.Update(msg)
		r.log = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case UpdateStatusBarMsg:
		cmds = append(cmds, r.updateStatusBarCmd)
	case tea.WindowSizeMsg:
		b, cmd := r.readme.Update(msg)
		r.readme = b.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, r.updateModels(msg))
	}
	t, cmd := r.tabs.Update(msg)
	r.tabs = t.(*tabs.Tabs)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	s, cmd := r.statusbar.Update(msg)
	r.statusbar = s.(*statusbar.StatusBar)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch r.activeTab {
	case readmeTab:
		b, cmd := r.readme.Update(msg)
		r.readme = b.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case filesTab:
		f, cmd := r.files.Update(msg)
		r.files = f.(*Files)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case commitsTab:
		l, cmd := r.log.Update(msg)
		r.log = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case branchesTab:
	case tagsTab:
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Repo) View() string {
	s := r.common.Styles.Repo.Copy().
		Width(r.common.Width).
		Height(r.common.Height)
	repoBodyStyle := r.common.Styles.RepoBody.Copy()
	hm := repoBodyStyle.GetVerticalFrameSize() +
		r.common.Styles.RepoHeader.GetHeight() +
		r.common.Styles.RepoHeader.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight() +
		r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	mainStyle := repoBodyStyle.
		Height(r.common.Height - hm)
	main := ""
	switch r.activeTab {
	case readmeTab:
		main = r.readme.View()
	case filesTab:
		main = r.files.View()
	case commitsTab:
		main = r.log.View()
	case branchesTab:
	case tagsTab:
	}
	view := lipgloss.JoinVertical(lipgloss.Top,
		r.headerView(),
		r.tabs.View(),
		mainStyle.Render(main),
		r.statusbar.View(),
	)
	return s.Render(view)
}

func (r *Repo) headerView() string {
	if r.selectedRepo == nil {
		return ""
	}
	name := r.common.Styles.RepoHeaderName.Render(r.selectedItem.Title())
	// TODO move this into a style.
	url := lipgloss.NewStyle().
		MarginLeft(1).
		Width(r.common.Width - lipgloss.Width(name) - 1).
		Align(lipgloss.Right).
		Render(r.selectedItem.URL())
	desc := r.common.Styles.RepoHeaderDesc.Render(r.selectedItem.Description())
	style := r.common.Styles.RepoHeader.Copy().Width(r.common.Width)
	return style.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.JoinHorizontal(lipgloss.Left,
				name,
				url,
			),
			desc,
		),
	)
}

func (r *Repo) setRepoCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		for _, r := range r.rs.AllRepos() {
			if r.Name() == repo {
				return RepoMsg(r)
			}
		}
		return common.ErrorMsg(git.ErrMissingRepo)
	}
}

func (r *Repo) updateStatusBarCmd() tea.Msg {
	value := ""
	info := ""
	switch r.activeTab {
	case readmeTab:
		info = fmt.Sprintf("%.f%%", r.readme.ScrollPercent()*100)
	case commitsTab:
		value = r.log.StatusBarValue()
		info = r.log.StatusBarInfo()
	case filesTab:
		value = r.files.StatusBarValue()
		info = r.files.StatusBarInfo()
	}
	return statusbar.StatusBarMsg{
		Key:    r.selectedRepo.Name(),
		Value:  value,
		Info:   info,
		Branch: fmt.Sprintf(" %s", r.ref.Name().Short()),
	}
}

func (r *Repo) updateReadmeCmd() tea.Msg {
	if r.selectedRepo == nil {
		return common.ErrorCmd(git.ErrMissingRepo)
	}
	rm, rp := r.selectedRepo.Readme()
	return r.readme.SetContent(rm, rp)
}

func (r *Repo) updateRefCmd() tea.Msg {
	head, err := r.selectedRepo.HEAD()
	if err != nil {
		return common.ErrorMsg(err)
	}
	return RefMsg(head)
}

func (r *Repo) updateModels(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	l, cmd := r.log.Update(msg)
	r.log = l.(*Log)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	f, cmd := r.files.Update(msg)
	r.files = f.(*Files)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func updateStatusBarCmd() tea.Msg {
	return UpdateStatusBarMsg{}
}
