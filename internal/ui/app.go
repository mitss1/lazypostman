package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/environment"
	"github.com/mitss1/lazypostman/internal/httpclient"
	"github.com/mitss1/lazypostman/internal/postman"
)

// Panel focus
const (
	PanelTree = iota
	PanelRequest
	PanelResponse
)

// Messages
type responseMsg struct {
	resp *httpclient.Response
}

type collectionsLoadedMsg struct {
	collections []postman.CollectionInfo
	err         error
}

type environmentsLoadedMsg struct {
	environments []postman.EnvironmentInfo
	err          error
}

type collectionFetchedMsg struct {
	collection *collection.Collection
	err        error
}

type environmentFetchedMsg struct {
	env *environment.Environment
	err error
}

type loginResultMsg struct {
	user *postman.UserInfo
	err  error
}

// App is the main TUI model
type App struct {
	collection *collection.Collection
	envManager *environment.Manager
	client     *httpclient.Client
	tree       TreeModel
	detail     DetailModel
	browser    BrowserModel
	login      LoginModel
	editor     EditorModel
	envPanel   EnvPanelModel
	focus      int
	width      int
	height     int
	loading    bool
	statusMsg  string
	maximized  int // -1 = none, 0=tree, 1=request, 2=response

	// Postman Cloud
	pmClient *postman.Client
	pmConfig *postman.Config
}

func NewApp(col *collection.Collection, envMgr *environment.Manager) App {
	client := httpclient.New(envMgr)
	tree := NewTreeModel(col)
	detail := NewDetailModel()

	envName := "No environment"
	if env := envMgr.ActiveEnvironment(); env != nil {
		envName = env.Name
	}

	// Try to load Postman config
	cfg, _ := postman.LoadConfig()
	var pmClient *postman.Client
	if cfg != nil && cfg.IsLoggedIn() {
		pmClient = postman.NewClient(cfg.APIKey)
	}

	return App{
		collection: col,
		envManager: envMgr,
		client:     client,
		tree:       tree,
		detail:     detail,
		browser:    NewBrowserModel(),
		login:      NewLoginModel(),
		editor:     NewEditorModel(),
		envPanel:   NewEnvPanelModel(envMgr),
		statusMsg:  fmt.Sprintf("Env: %s", envName),
		maximized:  -1,
		pmClient:   pmClient,
		pmConfig:   cfg,
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.browser.SetSize(msg.Width, msg.Height)
		a.login.SetSize(msg.Width, msg.Height)
		a.editor.SetSize(msg.Width, msg.Height)
		a.envPanel.SetSize(msg.Width, msg.Height)
		a.updateSizes()
		return a, nil

	case tea.KeyMsg:
		// Overlays intercept keys first
		if a.editor.IsVisible() {
			return a.handleEditorKey(msg)
		}
		if a.envPanel.IsVisible() {
			return a.handleEnvPanelKey(msg)
		}
		if a.login.IsVisible() {
			return a.handleLoginKey(msg)
		}
		if a.browser.IsVisible() {
			return a.handleBrowserKey(msg)
		}
		return a.handleKey(msg)

	case responseMsg:
		a.loading = false
		a.detail.SetResponse(msg.resp)
		if msg.resp.Error != nil {
			a.statusMsg = fmt.Sprintf("Error: %s", msg.resp.Error)
		} else {
			a.statusMsg = fmt.Sprintf("%s  %dms  %s",
				msg.resp.Status,
				msg.resp.Duration.Milliseconds(),
				formatSize(msg.resp.Size),
			)
		}
		return a, nil

	case loginResultMsg:
		if msg.err != nil {
			a.login.SetError(msg.err.Error())
			return a, nil
		}
		a.login.Hide()
		a.statusMsg = fmt.Sprintf("Logged in as %s", msg.user.FullName)
		return a, nil

	case collectionsLoadedMsg:
		if msg.err != nil {
			a.browser.SetError(msg.err)
			return a, nil
		}
		a.browser.SetCollections(msg.collections)
		return a, nil

	case environmentsLoadedMsg:
		if msg.err != nil {
			a.browser.SetError(msg.err)
			return a, nil
		}
		a.browser.SetEnvironments(msg.environments)
		return a, nil

	case collectionFetchedMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("Error: %s", msg.err)
			return a, nil
		}
		a.browser.Hide()
		a.collection = msg.collection
		a.tree = NewTreeModel(msg.collection)
		a.detail = NewDetailModel()
		a.updateSizes()
		a.statusMsg = fmt.Sprintf("Loaded: %s", msg.collection.Info.Name)
		return a, nil

	case environmentFetchedMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("Error: %s", msg.err)
			return a, nil
		}
		a.browser.Hide()
		a.envManager.AddEnvironment(msg.env)
		a.statusMsg = fmt.Sprintf("Loaded env: %s", msg.env.Name)
		return a, nil
	}

	return a, nil
}

// --- Login key handling ---

func (a App) handleLoginKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle bracketed paste (Cmd+V / Ctrl+Shift+V)
	if msg.Paste {
		a.login.Paste(string(msg.Runes))
		return a, nil
	}

	switch msg.String() {
	case "esc":
		a.login.Hide()
		return a, nil
	case "enter":
		apiKey := strings.TrimSpace(a.login.Value())
		if apiKey == "" {
			return a, nil
		}
		// Verify API key
		client := postman.NewClient(apiKey)
		return a, func() tea.Msg {
			user, err := client.GetMe()
			if err != nil {
				return loginResultMsg{err: err}
			}
			// Save config
			cfg := &postman.Config{APIKey: apiKey}
			_ = postman.SaveConfig(cfg)
			return loginResultMsg{user: user}
		}
	case "backspace", "ctrl+h":
		a.login.Backspace()
		return a, nil
	default:
		if len(msg.Runes) == 1 {
			a.login.TypeChar(msg.Runes[0])
		}
		return a, nil
	}
}

// --- Browser key handling ---

func (a App) handleBrowserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		a.browser.Hide()
		return a, nil
	case "j", "down":
		a.browser.MoveDown()
		return a, nil
	case "k", "up":
		a.browser.MoveUp()
		return a, nil
	case "enter":
		return a.browserSelect()
	}
	return a, nil
}

func (a App) browserSelect() (tea.Model, tea.Cmd) {
	if a.pmClient == nil {
		return a, nil
	}
	client := a.pmClient

	switch a.browser.mode {
	case BrowseCollections:
		uid := a.browser.SelectedCollectionUID()
		if uid == "" {
			return a, nil
		}
		a.browser.SetLoading(true)
		a.statusMsg = "Fetching collection..."
		return a, func() tea.Msg {
			col, err := client.GetCollection(uid)
			return collectionFetchedMsg{collection: col, err: err}
		}
	case BrowseEnvironments:
		uid := a.browser.SelectedEnvironmentUID()
		if uid == "" {
			return a, nil
		}
		a.browser.SetLoading(true)
		a.statusMsg = "Fetching environment..."
		return a, func() tea.Msg {
			env, err := client.GetEnvironment(uid)
			return environmentFetchedMsg{env: env, err: err}
		}
	}
	return a, nil
}

// --- Main key handling ---

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit

	case "tab":
		a.focus = (a.focus + 1) % 3
		return a, nil

	case "shift+tab":
		a.focus = (a.focus + 2) % 3
		return a, nil

	case "j", "down":
		switch a.focus {
		case PanelTree:
			a.tree.MoveDown()
			a.updateSelectedRequest()
		case PanelRequest:
			a.detail.ScrollReqDown(a.detail.ReqContentLines())
		case PanelResponse:
			a.detail.ScrollRespDown(a.detail.RespContentLines())
		}
		return a, nil

	case "k", "up":
		switch a.focus {
		case PanelTree:
			a.tree.MoveUp()
			a.updateSelectedRequest()
		case PanelRequest:
			a.detail.ScrollReqUp()
		case PanelResponse:
			a.detail.ScrollRespUp()
		}
		return a, nil

	case "l", "right":
		if a.focus == PanelTree {
			a.tree.Toggle()
		}
		return a, nil

	case "h", "left":
		if a.focus == PanelTree {
			a.tree.Toggle()
		}
		return a, nil

	case " ":
		if a.focus == PanelTree {
			a.tree.Toggle()
		}
		return a, nil

	case "enter":
		return a.sendRequest()

	case "t":
		if a.focus == PanelRequest {
			a.detail.NextTab()
		} else if a.focus == PanelResponse {
			a.detail.NextRespTab()
		}
		return a, nil

	case "e":
		a.cycleEnvironment()
		return a, nil

	case "o":
		return a.openCollectionBrowser()

	case "E":
		return a.openEnvironmentBrowser()

	case "L":
		if a.pmConfig != nil && a.pmConfig.IsLoggedIn() {
			a.statusMsg = "Already logged in. Config: ~/.config/lazypostman/config.json"
		} else {
			a.login.Show()
		}
		return a, nil

	case "+", "=":
		if a.maximized == -1 {
			a.maximized = a.focus
			a.updateSizes()
		}
		return a, nil

	case "-", "_":
		if a.maximized != -1 {
			a.maximized = -1
			a.updateSizes()
		}
		return a, nil

	case "g":
		switch a.focus {
		case PanelRequest:
			a.detail.ResetReqScroll()
		case PanelResponse:
			a.detail.ResetRespScroll()
		}
		return a, nil

	case "G":
		switch a.focus {
		case PanelRequest:
			a.detail.ScrollReqToBottom()
		case PanelResponse:
			a.detail.ScrollRespToBottom()
		}
		return a, nil

	case "i":
		return a.openEditor()

	case "v":
		a.envPanel.SetSize(a.width, a.height)
		a.envPanel.Show()
		return a, nil

	case "?":
		if a.statusMsg == helpText() {
			a.statusMsg = ""
		} else {
			a.statusMsg = helpText()
		}
		return a, nil
	}

	return a, nil
}

func (a *App) openCollectionBrowser() (tea.Model, tea.Cmd) {
	if a.pmClient == nil {
		if a.pmConfig == nil || !a.pmConfig.IsLoggedIn() {
			a.login.Show()
			return a, nil
		}
		a.pmClient = postman.NewClient(a.pmConfig.APIKey)
	}

	a.browser.Show(BrowseCollections)
	a.browser.SetLoading(true)
	client := a.pmClient
	return a, func() tea.Msg {
		cols, err := client.ListCollections()
		return collectionsLoadedMsg{collections: cols, err: err}
	}
}

func (a *App) openEnvironmentBrowser() (tea.Model, tea.Cmd) {
	if a.pmClient == nil {
		if a.pmConfig == nil || !a.pmConfig.IsLoggedIn() {
			a.login.Show()
			return a, nil
		}
		a.pmClient = postman.NewClient(a.pmConfig.APIKey)
	}

	a.browser.Show(BrowseEnvironments)
	a.browser.SetLoading(true)
	client := a.pmClient
	return a, func() tea.Msg {
		envs, err := client.ListEnvironments()
		return environmentsLoadedMsg{environments: envs, err: err}
	}
}

func (a *App) updateSelectedRequest() {
	if req := a.tree.SelectedRequest(); req != nil {
		a.detail.SetRequest(req)
	}
}

func (a *App) sendRequest() (tea.Model, tea.Cmd) {
	req := a.tree.SelectedRequest()
	if req == nil {
		a.statusMsg = "No request selected"
		return a, nil
	}

	a.loading = true
	a.statusMsg = "Sending..."

	client := a.client
	return a, func() tea.Msg {
		resp := client.Execute(req)
		return responseMsg{resp: resp}
	}
}

func (a *App) cycleEnvironment() {
	envs := a.envManager.Environments()
	if len(envs) == 0 {
		a.statusMsg = "No environments loaded"
		return
	}

	current := -1
	if env := a.envManager.ActiveEnvironment(); env != nil {
		for i, e := range envs {
			if e.Name == env.Name {
				current = i
				break
			}
		}
	}

	next := (current + 1) % len(envs)
	a.envManager.SetActive(next)
	a.statusMsg = fmt.Sprintf("Env: %s", envs[next].Name)
}

// --- Editor key handling ---

func (a App) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Paste {
		a.editor.Paste(string(msg.Runes))
		return a, nil
	}

	switch msg.String() {
	case "esc":
		a.editor.Close()
		return a, nil
	case "enter":
		if a.editor.mode == EditBody {
			a.editor.NewLine()
			return a, nil
		}
		a.saveEditor()
		a.editor.Close()
		return a, nil
	case "ctrl+s":
		a.saveEditor()
		a.editor.Close()
		return a, nil
	case "backspace", "ctrl+h":
		a.editor.Backspace()
		return a, nil
	case "delete":
		a.editor.Delete()
		return a, nil
	case "left":
		a.editor.MoveLeft()
		return a, nil
	case "right":
		a.editor.MoveRight()
		return a, nil
	case "home":
		a.editor.Home()
		return a, nil
	case "end":
		a.editor.End()
		return a, nil
	default:
		if len(msg.Runes) == 1 {
			a.editor.TypeChar(msg.Runes[0])
		}
		return a, nil
	}
}

func (a *App) openEditor() (tea.Model, tea.Cmd) {
	req := a.tree.SelectedRequest()

	switch a.focus {
	case PanelRequest:
		if req == nil {
			a.statusMsg = "No request selected"
			return a, nil
		}
		switch a.detail.tab {
		case 0: // Params
			if len(req.URL.Query) > 0 {
				q := req.URL.Query[0]
				a.editor.Open(EditParamValue, fmt.Sprintf("Param: %s", q.Key), q.Value, 0)
			} else {
				a.statusMsg = "No params to edit"
			}
		case 1: // Headers
			if len(req.Header) > 0 {
				h := req.Header[0]
				a.editor.Open(EditHeaderValue, fmt.Sprintf("Header: %s", h.Key), h.Value, 0)
			} else {
				a.statusMsg = "No headers to edit"
			}
		case 2: // Body
			raw := ""
			if req.Body != nil {
				raw = req.Body.Raw
			}
			a.editor.Open(EditBody, "Request Body", raw, 0)
		}
	case PanelTree:
		if req == nil {
			a.statusMsg = "No request selected"
			return a, nil
		}
		a.editor.Open(EditURL, "URL", req.URL.Raw, 0)
	}

	return a, nil
}

func (a *App) saveEditor() {
	req := a.tree.SelectedRequest()
	value := a.editor.Value()

	switch a.editor.Mode() {
	case EditURL:
		if req != nil {
			req.URL.Raw = value
		}
	case EditParamValue:
		if req != nil && a.editor.Index() < len(req.URL.Query) {
			req.URL.Query[a.editor.Index()].Value = value
		}
	case EditHeaderValue:
		if req != nil && a.editor.Index() < len(req.Header) {
			req.Header[a.editor.Index()].Value = value
		}
	case EditBody:
		if req != nil {
			if req.Body == nil {
				req.Body = &collection.Body{Mode: "raw"}
			}
			req.Body.Raw = value
		}
	case EditEnvValue:
		a.envPanel.UpdateVar(a.editor.Index(), value)
	}

	a.statusMsg = "Saved"
}

// --- Env panel key handling ---

func (a App) handleEnvPanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "v":
		a.envPanel.Hide()
		return a, nil
	case "j", "down":
		a.envPanel.MoveDown()
		return a, nil
	case "k", "up":
		a.envPanel.MoveUp()
		return a, nil
	case " ":
		a.envPanel.ToggleSelected()
		return a, nil
	case "e":
		a.cycleEnvironment()
		a.envPanel.Show() // refresh after cycle
		return a, nil
	case "enter":
		key, value, index := a.envPanel.SelectedVar()
		if index >= 0 {
			a.editor.Open(EditEnvValue, fmt.Sprintf("Env: %s", key), value, index)
		}
		return a, nil
	}
	return a, nil
}

func (a *App) updateSizes() {
	panelHeight := a.height - 4

	if a.maximized >= 0 {
		a.tree.SetSize(a.width-2, panelHeight)
		a.detail.SetSize(a.width-2, panelHeight)
		return
	}

	treeWidth := a.width * 30 / 100
	if treeWidth < 25 {
		treeWidth = 25
	}
	detailWidth := a.width - treeWidth - 3

	a.tree.SetSize(treeWidth, panelHeight)
	a.detail.SetSize(detailWidth, panelHeight/2)
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Overlay views take over the screen
	if a.editor.IsVisible() {
		overlay := a.renderOverlay(a.editor.View())
		return lipgloss.JoinVertical(lipgloss.Left, overlay, a.renderStatusBar())
	}

	if a.envPanel.IsVisible() {
		overlay := a.renderOverlay(a.envPanel.View())
		return lipgloss.JoinVertical(lipgloss.Left, overlay, a.renderStatusBar())
	}

	if a.login.IsVisible() {
		overlay := a.renderOverlay(a.login.View())
		return lipgloss.JoinVertical(lipgloss.Left, overlay, a.renderStatusBar())
	}

	if a.browser.IsVisible() {
		overlay := a.renderOverlay(a.browser.View())
		return lipgloss.JoinVertical(lipgloss.Left, overlay, a.renderStatusBar())
	}

	panelHeight := a.height - 4

	// Styles
	activeBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3465a4"))

	inactiveBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555"))

	// Handle maximized panel
	if a.maximized >= 0 {
		maxBorder := activeBorder.Width(a.width - 2).Height(panelHeight)
		var content string
		switch a.maximized {
		case PanelTree:
			treeTitle := titleStyle("  "+a.collection.Info.Name+" ") + dimText("  (press - to restore)")
			content = maxBorder.Render(treeTitle + "\n" + a.tree.View())
		case PanelRequest:
			reqTitle := titleStyle(" Request ") + dimText("  (press - to restore)")
			content = maxBorder.Render(reqTitle + "\n" + a.detail.RequestView())
		case PanelResponse:
			respTitle := titleStyle(" Response ") + dimText("  (press - to restore)")
			if a.loading {
				respTitle = titleStyle(" Response ⏳ ") + dimText("  (press - to restore)")
			}
			content = maxBorder.Render(respTitle + "\n" + a.detail.ResponseView())
		}
		return lipgloss.JoinVertical(lipgloss.Left, content, a.renderStatusBar())
	}

	// Normal layout
	treeWidth := a.width * 30 / 100
	if treeWidth < 25 {
		treeWidth = 25
	}
	detailWidth := a.width - treeWidth - 3

	// Tree panel
	treeBorder := inactiveBorder
	if a.focus == PanelTree {
		treeBorder = activeBorder
	}
	treeTitle := titleStyle("  " + a.collection.Info.Name + " ")
	treePanel := treeBorder.
		Width(treeWidth).
		Height(panelHeight).
		MaxHeight(panelHeight + 2).
		Render(treeTitle + "\n" + a.tree.View())

	// Request panel
	reqBorder := inactiveBorder
	if a.focus == PanelRequest {
		reqBorder = activeBorder
	}
	reqHeight := panelHeight / 2
	reqTitle := titleStyle(" Request ")
	reqPanel := reqBorder.
		Width(detailWidth).
		Height(reqHeight).
		MaxHeight(reqHeight + 2).
		Render(reqTitle + "\n" + a.detail.RequestView())

	// Response panel
	respBorder := inactiveBorder
	if a.focus == PanelResponse {
		respBorder = activeBorder
	}
	respHeight := panelHeight - reqHeight - 2
	respTitle := titleStyle(" Response ")
	if a.loading {
		respTitle = titleStyle(" Response ⏳ ")
	}
	respPanel := respBorder.
		Width(detailWidth).
		Height(respHeight).
		MaxHeight(respHeight + 2).
		Render(respTitle + "\n" + a.detail.ResponseView())

	// Compose layout
	rightSide := lipgloss.JoinVertical(lipgloss.Left, reqPanel, respPanel)
	main := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, rightSide)

	// Status bar
	statusBar := a.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func (a App) renderOverlay(content string) string {
	panelHeight := a.height - 4
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3465a4")).
		Width(a.width - 2).
		Height(panelHeight).
		MaxHeight(panelHeight + 2)
	return overlayStyle.Render(content)
}

func (a App) renderStatusBar() string {
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a2e")).
		Foreground(lipgloss.Color("#888888")).
		Width(a.width).
		Padding(0, 1)

	left := a.statusMsg

	// Show login status
	right := "?=help  q=quit"
	if a.pmConfig != nil && a.pmConfig.IsLoggedIn() {
		right = "●=connected  " + right
	}

	spaces := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if spaces < 1 {
		spaces = 1
	}

	return statusStyle.Render(left + strings.Repeat(" ", spaces) + right)
}

func titleStyle(s string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3465a4")).
		Padding(0, 1).
		Render(s)
}

func helpText() string {
	return "j/k=scroll  tab=panel  enter=send  i=edit  v=env vars  o=collections  E=environments  L=login  +=max  -=restore  e=env  ?=help  q=quit"
}
