package main

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gizak/termui"
	"github.com/skratchdot/open-golang/open"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var monitoringCmd = cli.Command{
	Name:    "monitoring",
	Short:   "CDS monitoring",
	Aliases: []string{"ui"},
}

func monitoringRun(v cli.Values) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("cds UI crashed :(\n%s\n", r)
			termui.Close()
		}
	}()

	if err := termui.Init(); err != nil {
		return nil, err
	}
	defer termui.Close()

	ui := newTermui()
	ui.init()
	ui.render()

	go func() {
		for range time.NewTicker(2 * time.Second).C {
			if err := ui.loadData(); err != nil {
				panic(err)
			}
			ui.render()
		}
	}()

	termui.Loop()
	return nil, nil
}

func newTermui() *Termui {
	return &Termui{baseURL: "http://cds.ui/"}
}

// Termui wrapper designed for dashboard creation
type Termui struct {
	header, times *termui.Par

	selected         string
	queueTabSelected int
	statusSelected   []sdk.Status
	baseURL          string

	me                             *sdk.User
	status                         *sdk.MonitoringStatus
	elapsedStatus                  time.Duration
	workers                        []sdk.Worker
	elapsedWorkers                 time.Duration
	services                       []sdk.Service
	elapsedWorkerModels            time.Duration
	pipelineBuildJob               []sdk.PipelineBuildJob
	workflowNodeJobRun             []sdk.WorkflowNodeJobRun
	elapsedWorkflowNodeJobRun      time.Duration
	workflowNodeJobRunCount        sdk.WorkflowNodeJobRunCount
	elapsedWorkflowNodeJobRunCount time.Duration

	// monitoring
	queue                   *cli.ScrollableList
	statusHatcheriesWorkers *cli.ScrollableList
	statusServices          *cli.ScrollableList
	currentURL              string
}

func (ui *Termui) loadData() error {
	urlUI, err := client.ConfigUser()
	if err != nil {
		return err
	}
	if b, ok := urlUI[sdk.ConfigURLUIKey]; ok {
		ui.baseURL = b
	}

	ui.me, err = client.UserGet(cfg.User)
	if err != nil {
		return fmt.Errorf("Can't get current user: %v", err)
	}

	start := time.Now()
	ui.status, err = client.MonStatus()
	if err != nil {
		return err
	}
	ui.elapsedStatus = time.Since(start)

	start = time.Now()
	ui.workers, err = client.WorkerList()
	if err != nil {
		return err
	}
	ui.elapsedWorkers = time.Since(start)

	if ui.me.Admin {
		ui.services, err = client.ServicesByType("hatchery")
		if err != nil {
			return err
		}
	}

	start = time.Now()
	if _, err := client.WorkerModels(); err != nil {
		return err
	}
	ui.elapsedWorkerModels = time.Since(start)

	ui.pipelineBuildJob, err = client.QueuePipelineBuildJob()
	if err != nil {
		return err
	}

	if err := ui.loadQueue(); err != nil {
		return err
	}

	start = time.Now()
	ui.workflowNodeJobRunCount, err = client.QueueCountWorkflowNodeJobRun(nil, nil)
	if err != nil {
		return err
	}
	ui.elapsedWorkflowNodeJobRunCount = time.Since(start)

	return nil
}

func (ui *Termui) loadQueue() error {
	switch ui.queueTabSelected {
	case 0:
		ui.statusSelected = []sdk.Status{sdk.StatusWaiting}
	case 1:
		ui.statusSelected = []sdk.Status{sdk.StatusBuilding}
	case 2:
		ui.statusSelected = []sdk.Status{sdk.StatusWaiting, sdk.StatusBuilding}
	}

	var err error

	start := time.Now()
	ui.workflowNodeJobRun, err = client.QueueWorkflowNodeJobRun(ui.statusSelected...)
	if err != nil {
		return err
	}
	ui.elapsedWorkflowNodeJobRun = time.Since(start)

	return nil
}

// Constants for each view of cds ui
const (
	QueueSelected = "queue"
)

func (ui *Termui) init() {
	// init termui handlers
	termui.Handle("/timer/1s", func(e termui.Event) {})
	termui.Handle("/sys/kbd/q", func(termui.Event) { termui.StopLoop() })
	termui.Handle("/sys/kbd", func(e termui.Event) { /*ui.msg = fmt.Sprintf("No command for %v", e)*/ })
	termui.Handle("/sys/kbd/<down>", func(e termui.Event) { ui.moveDown() })
	termui.Handle("/sys/kbd/<up>", func(e termui.Event) { ui.moveUp() })
	termui.Handle("/sys/kbd/<left>", func(e termui.Event) { ui.moveLeft() })
	termui.Handle("/sys/kbd/<right>", func(e termui.Event) { ui.moveRight() })
	termui.Handle("/sys/kbd/<enter>", func(e termui.Event) {
		if ui.currentURL != "" {
			open.Run(ui.currentURL)
		}
	})

	ui.header = newPar()
	ui.times = newPar()

	ui.selected = QueueSelected

	// prepare queue list
	ui.queue = cli.NewScrollableList()
	ui.queue.ItemFgColor = termui.ColorWhite
	ui.queue.ItemBgColor = termui.ColorBlack
	ui.queue.BorderLabel = " Queue "
	ui.queue.Height = int(math.Max(float64(termui.TermHeight()-heightBottom), 4))
	ui.queue.Width = termui.TermWidth()
	ui.queue.Items = []string{"Loading..."}
	ui.queue.BorderBottom = false
	ui.queue.BorderLeft = false
	ui.queue.BorderRight = false

	// prepare list of hatcheries and workers status
	ui.statusHatcheriesWorkers = cli.NewScrollableList()
	ui.statusHatcheriesWorkers.BorderLabel = " Hatcheries "
	ui.statusHatcheriesWorkers.Height = heightBottom
	ui.statusHatcheriesWorkers.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.statusHatcheriesWorkers.BorderBottom = false
	ui.statusHatcheriesWorkers.BorderLeft = true
	ui.statusHatcheriesWorkers.BorderRight = false

	// prepare services status list
	ui.statusServices = cli.NewScrollableList()
	ui.statusServices.BorderLabel = " Status "
	ui.statusServices.Height = heightBottom
	ui.statusServices.Items = []string{"[loading...](fg-cyan,bg-default)"}
	ui.statusServices.BorderBottom = false
	ui.statusServices.BorderLeft = false
	ui.statusServices.BorderRight = false

	termui.Body.Rows = nil
	termui.Body.AddRows(
		termui.NewRow(termui.NewCol(12, 0, ui.header)),
		termui.NewRow(termui.NewCol(12, 0, ui.times)),
	)
	termui.Body.AddRows(termui.NewCol(12, 0, ui.queue))
	termui.Body.AddRows(termui.NewRow(
		termui.NewCol(7, 0, ui.statusServices),
		termui.NewCol(5, 0, ui.statusHatcheriesWorkers),
	))
}

func newPar() *termui.Par {
	p := termui.NewPar("")
	p.Height = 1
	p.TextFgColor = termui.ColorWhite
	p.BorderLabel = ""
	p.BorderFg = termui.ColorCyan
	p.Border = false
	return p
}

const (
	heightBottom int = 25
)

func (ui *Termui) render() {
	checking, checkingColor := statusShort(sdk.StatusChecking.String())
	waiting, waitingColor := statusShort(sdk.StatusWaiting.String())
	building, buildingColor := statusShort(sdk.StatusBuilding.String())
	success, successColor := statusShort(sdk.StatusSuccess.String())
	fail, failColor := statusShort(sdk.StatusFail.String())
	disabled, disabledColor := statusShort(sdk.StatusDisabled.String())
	ui.header.Text = fmt.Sprintf("[CDS | (q)uit | Legend: ](fg-cyan)[Checking:%s](%s)  [Waiting:%s](%s)  [Building:%s](%s)  [Success:%s](%s)  [Fail:%s](%s)  [Disabled:%s](%s)",
		checking, checkingColor,
		waiting, waitingColor,
		building, buildingColor,
		success, successColor,
		fail, failColor,
		disabled, disabledColor)

	ui.times.Text = fmt.Sprintf(
		"[count queue wf %s](fg-cyan,bg-default) | [queue wf %s](fg-cyan,bg-default) | [workers %s](fg-cyan,bg-default) | [wModels %s](fg-cyan,bg-default) | [status %s](fg-cyan,bg-default)",
		sdk.Round(ui.elapsedWorkflowNodeJobRunCount, time.Millisecond).String(),
		sdk.Round(ui.elapsedWorkflowNodeJobRun, time.Millisecond).String(),
		sdk.Round(ui.elapsedWorkers, time.Millisecond).String(),
		sdk.Round(ui.elapsedWorkerModels, time.Millisecond).String(),
		sdk.Round(ui.elapsedStatus, time.Millisecond).String(),
	)
	//ui.msg = fmt.Sprintf("[%s](bg-red)", err.Error())

	ui.monitoringColorSelected()

	ui.updateQueue(ui.baseURL)
	ui.computeStatusHatcheriesWorkers(ui.workers)
	ui.updateStatus()

	termui.Body.Align()
	termui.Render(termui.Body)
	termui.Render()
}

func (ui *Termui) moveDown() {
	switch ui.selected {
	case QueueSelected:
		ui.queue.CursorDown()
	}
	ui.render()
}

func (ui *Termui) moveUp() {
	switch ui.selected {
	case QueueSelected:
		ui.queue.CursorUp()
	}
	ui.render()
}

func (ui *Termui) moveLeft() {
	switch ui.selected {
	case QueueSelected:
		ui.decrementQueueFilter()
	}
	ui.render()
}

func (ui *Termui) moveRight() {
	switch ui.selected {
	case QueueSelected:
		ui.incrementQueueFilter()
	}
	ui.render()
}

func (ui *Termui) incrementQueueFilter() {
	if ui.queueTabSelected < 2 {
		ui.queueTabSelected++
	} else {
		ui.queueTabSelected = 0
	}
	if err := ui.loadQueue(); err != nil {
		panic(err)
	}
}

func (ui *Termui) decrementQueueFilter() {
	if 0 < ui.queueTabSelected {
		ui.queueTabSelected--
	} else {
		ui.queueTabSelected = 2
	}
	if err := ui.loadQueue(); err != nil {
		panic(err)
	}
}

func (ui *Termui) monitoringColorSelected() {
	ui.queue.BorderFg = termui.ColorDefault
	ui.statusHatcheriesWorkers.BorderFg = termui.ColorDefault
	ui.statusServices.BorderFg = termui.ColorDefault

	switch ui.selected {
	case QueueSelected:
		ui.queue.BorderFg = termui.ColorRed
	}

	termui.Render(ui.queue, ui.statusHatcheriesWorkers, ui.statusServices)
}

func (ui *Termui) updateStatus() {
	items := []string{}
	if ui.status != nil {
		for _, l := range ui.status.Lines {
			if l.Status == sdk.MonitoringStatusWarn {
				items = append(items, fmt.Sprintf("[%s](fg-yellow,bg-default)", l.String()))
			} else if l.Status != sdk.MonitoringStatusOK {
				items = append(items, fmt.Sprintf("[%s](fg-white,bg-red)", l.String()))
			} else if strings.Contains(l.Component, "Global") {
				items = append(items, fmt.Sprintf("[%s](fg-white,bg-default)", l.String()))
			}
		}
	}
	ui.statusServices.Items = items
}

func (ui *Termui) computeStatusHatcheriesWorkers(workers []sdk.Worker) {
	hatcheryNames, statusTitle := []string{}, []string{}
	hatcheries := make(map[string]map[string]int64)
	status := make(map[string]int)

	if ui.me != nil && ui.me.Admin {
		for _, s := range ui.services {
			if _, ok := hatcheries[s.Name]; !ok {
				hatcheries[s.Name] = make(map[string]int64)
				hatcheryNames = append(hatcheryNames, s.Name)
			}
		}
	}

	without := "Without hatchery"
	hatcheries[without] = make(map[string]int64)
	hatcheryNames = append(hatcheryNames, without)

	for _, w := range workers {
		var name string
		if w.HatcheryID == 0 {
			name = "Without hatchery"
		} else {
			name = w.HatcheryName
		}
		if _, ok := hatcheries[name]; !ok {
			hatcheries[name] = make(map[string]int64)
			hatcheryNames = append(hatcheryNames, name)
		}
		hatcheries[name][w.Status.String()] = hatcheries[name][w.Status.String()] + 1
		if _, ok := status[w.Status.String()]; !ok {
			statusTitle = append(statusTitle, w.Status.String())
		}
		status[w.Status.String()] = status[w.Status.String()] + 1
	}

	items := []string{}
	sort.Strings(hatcheryNames)
	for _, name := range hatcheryNames {
		v := hatcheries[name]
		var t string
		for _, status := range statusTitle {
			if v[status] > 0 {
				icon, color := statusShort(status)
				t += fmt.Sprintf("[ %d %s ](%s,bg-default)", v[status], icon, color)
			}
		}
		if len(t) == 0 {
			t += fmt.Sprintf("[ _ ](fg-white,bg-default)")
		}
		t += fmt.Sprintf("[ %s](fg-white,bg-default)", name)
		items = append(items, t)
	}
	ui.statusHatcheriesWorkers.Items = items

	sort.Strings(statusTitle)
	title := " Hatcheries "
	for _, s := range statusTitle {
		icon, color := statusShort(s)
		title += fmt.Sprintf("[%d %s](%s) ", status[s], icon, color)
	}
	ui.statusHatcheriesWorkers.BorderLabel = title
}

func (ui *Termui) updateQueue(baseURL string) {
	var maxQueued time.Duration

	items := []string{
		fmt.Sprintf("[  _ %s %s%s %s ➤ %s ➤ %s ➤ %s](fg-cyan,bg-default)",
			pad("since", 9), pad("by", 27), pad("run", 7), pad("project/workflow", 30),
			pad("node", 20), pad("triggered by", 17), "requirements"),
	}

	var idx int
	var item string
	for _, job := range ui.pipelineBuildJob {
		item, maxQueued = ui.updateQueueJob(idx, maxQueued, job.ID, false, job.Parameters,
			job.Job, job.Queued, job.BookedBy, baseURL, job.Status)
		items = append(items, item)
		idx++
	}

	for _, job := range ui.workflowNodeJobRun {
		item, maxQueued = ui.updateQueueJob(idx, maxQueued, job.ID, true, job.Parameters,
			job.Job, job.Queued, job.BookedBy, baseURL, job.Status)
		items = append(items, item)
		idx++
	}
	ui.queue.Items = items

	ui.queue.BorderLabel = fmt.Sprintf("Queue(%s):%d - Max Waiting:%s ", fmt.Sprintf("%v", ui.statusSelected),
		ui.workflowNodeJobRunCount.Count+int64(len(ui.pipelineBuildJob)),
		sdk.Round(maxQueued, time.Second).String())
}

func (ui *Termui) updateQueueJob(idx int, maxQueued time.Duration, id int64, isWJob bool, parameters []sdk.Parameter, executedJob sdk.ExecutedJob, queued time.Time, bookedBy sdk.Hatchery, baseURL, status string) (string, time.Duration) {
	req := ""
	for _, r := range executedJob.Job.Action.Requirements {
		req += fmt.Sprintf("%s:%s ", r.Type, r.Value)
	}
	prj := getVarsInPbj("cds.project", parameters)
	app := getVarsInPbj("cds.application", parameters)
	pip := getVarsInPbj("cds.pipeline", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	node := getVarsInPbj("cds.node", parameters)
	run := getVarsInPbj("cds.run", parameters)
	runNumber := getVarsInPbj("cds.run.number", parameters)
	build := getVarsInPbj("cds.buildNumber", parameters)
	env := getVarsInPbj("cds.environment", parameters)
	bra := getVarsInPbj("git.branch", parameters)
	version := getVarsInPbj("cds.version", parameters)
	triggeredBy := getVarsInPbj("cds.triggered_by.username", parameters)
	duration := time.Since(queued)
	var currentURL, fgColor string

	row := make([]string, 6)
	var c string
	if duration > 60*time.Second {
		c = "bg-red"
	} else if duration > 15*time.Second {
		c = "bg-yellow"
	} else {
		c = "bg-default"
	}

	row[0] = pad(fmt.Sprintf(sdk.Round(duration, time.Second).String()), 9)
	if isWJob {
		fgColor = "fg-white"
		row[2] = pad(fmt.Sprintf("%s", run), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+workflow, 30), pad(node, 20))
		currentURL = fmt.Sprintf("%s/project/%s/workflow/%s/run/%s", baseURL, prj, workflow, runNumber)
	} else {
		row[2] = pad(fmt.Sprintf("%d", id), 7)
		row[3] = fmt.Sprintf("%s ➤ %s", pad(prj+"/"+app, 30), pad(pip+"/"+bra+"/"+env, 20))
		currentURL = fmt.Sprintf("%s/project/%s/application/%s/pipeline/%s/build/%s?envName=%s&branch=%s&version=%s",
			baseURL, prj, app, pip, build, url.QueryEscape(env), url.QueryEscape(bra), version,
		)
		fgColor = "fg-magenta"
	}

	if status == sdk.StatusBuilding.String() {
		row[1] = pad(fmt.Sprintf(" %s.%s ", executedJob.WorkerName, executedJob.WorkerID), 27)
	} else if bookedBy.ID != 0 {
		row[1] = pad(fmt.Sprintf(" %s.%d ", bookedBy.Name, bookedBy.ID), 27)
	} else {
		row[1] = pad("", 27)
	}

	row[4] = fmt.Sprintf("➤ %s", pad(triggeredBy, 17))
	row[5] = fmt.Sprintf("➤ %s", req)

	_, color := statusShort(status)
	color = strings.Replace(color, "fg", "bg", 1)
	item := fmt.Sprintf("  [ ](%s)[ ](bg-default)[%s](%s)[%s %s %s %s %s](%s,bg-default)", color, row[0], c, row[1], row[2], row[3], row[4], row[5], fgColor)

	if idx == ui.queue.Cursor-1 {
		ui.currentURL = currentURL
	}
	if maxQueued < duration {
		return item, duration
	}
	return item, maxQueued
}

func statusShort(status string) (string, string) {
	switch status {
	case sdk.StatusWaiting.String():
		return "☕", "fg-cyan"
	case sdk.StatusBuilding.String():
		return "▶", "fg-blue"
	case sdk.StatusDisabled.String():
		return "⏏", "fg-grey"
	case sdk.StatusChecking.String():
		return "♻", "fg-yellow"
	case sdk.StatusSuccess.String():
		return "✔", "fg-green"
	case sdk.StatusFail.String():
		return "✖", "fg-red"
	}
	return status, "fg-default"
}

func pad(t string, size int) string {
	if len(t) > size {
		return t[0:size-3] + "..."
	}
	return t + strings.Repeat(" ", size-len(t))
}

func getVarsInPbj(key string, ps []sdk.Parameter) string {
	for _, p := range ps {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}
