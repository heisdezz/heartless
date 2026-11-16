package screens

import (
	"github.com/egoist/mygo/ui"

	"heartless/internal/app"
	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
)

// scriptRunner runs post-response scripts off the frame loop. A script can
// take seconds, so it never runs while a frame is being built: the view
// queues it, and the runner hands the result back under the lock and asks
// for a new frame.
type scriptRunner struct {
	queue chan *scriptTask
}

// scriptTask is one queued run of a request's post-response script.
type scriptTask struct {
	rs       *requestState
	src      string
	request  *ScriptRequest
	response *domainres.Response
	env      ScriptEnvironment
	gen      uint64      // the run this belongs to, to drop stale results
	ctx      *ui.Context // asks for a frame when the result is ready
}

func newScriptRunner() *scriptRunner {
	r := &scriptRunner{queue: make(chan *scriptTask, 16)}
	go func() {
		for task := range r.queue {
			result := RunPostResponseScript(task.src, task.request, task.response, task.env)

			mu.Lock()
			// A newer run for the same request has started: its result is the
			// one the pane should show.
			if task.rs.scriptGen == task.gen {
				task.rs.script = &scriptRun{result: result}
			}
			mu.Unlock()

			if task.ctx != nil {
				task.ctx.Invalidate()
			}
		}
	}()
	return r
}

// enqueue schedules a run, dropping it when the queue is full so a burst of
// sends cannot back up.
func (r *scriptRunner) enqueue(task *scriptTask) {
	if r == nil || task == nil {
		return
	}
	select {
	case r.queue <- task:
	default:
	}
}

// queueScript starts a run for rs and returns at once. The result lands in
// rs.script when the runner finishes.
func (r *scriptRunner) queueScript(c *ui.Context, st *shellState, services *app.Services, req domainreq.Request, rs *requestState) {
	if r == nil || rs == nil || services == nil || rs.Script == "" {
		return
	}
	r.enqueue(st.newScriptTask(c, req, rs))
}

// newScriptTask captures everything a run needs, so later edits to the pane
// do not change a run already queued.
func (st *shellState) newScriptTask(c *ui.Context, req domainreq.Request, rs *requestState) *scriptTask {
	sent := rs.buildDomainRequest(req)
	rs.scriptGen++
	return &scriptTask{
		rs:       rs,
		src:      rs.Script,
		request:  scriptRequestFromDomain(*sent),
		response: rs.responseOf(),
		env:      st.scriptEnv(),
		gen:      rs.scriptGen,
		ctx:      c,
	}
}
