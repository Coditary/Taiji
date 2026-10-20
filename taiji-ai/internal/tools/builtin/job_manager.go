package builtin

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	jobRingMaxBytes = 64 * 1024
	jobRingMaxLines = 500
)

var defaultJobManager = NewJobManager()

// DefaultJobManager returns the process-wide background job manager.
func DefaultJobManager() *JobManager {
	return defaultJobManager
}

// JobManager tracks background shell jobs per chat session.
type JobManager struct {
	mu       sync.Mutex
	sessions map[string]*sessionJobs
	seq      uint64
}

type sessionJobs struct {
	mu   sync.Mutex
	jobs map[string]*backgroundJob
}

type backgroundJob struct {
	id       string
	session  string
	command  string
	cwd      string
	pid      int
	pgid     int
	started  time.Time
	buffer   *lineRingBuffer
	done     chan struct{}
	exitCode int
	running  bool
}

// NewJobManager creates an empty job manager.
func NewJobManager() *JobManager {
	return &JobManager{sessions: map[string]*sessionJobs{}}
}

func (m *JobManager) Start(sessionID, command, cwd string) (string, int, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", 0, fmt.Errorf("command is required")
	}
	if sessionID == "" {
		sessionID = "default"
	}
	m.mu.Lock()
	m.seq++
	id := fmt.Sprintf("job-%d", m.seq)
	m.mu.Unlock()

	job := &backgroundJob{
		id:      id,
		session: sessionID,
		command: command,
		cwd:     strings.TrimSpace(cwd),
		started: time.Now(),
		buffer:  newLineRingBuffer(jobRingMaxBytes, jobRingMaxLines),
		done:    make(chan struct{}),
		running: true,
	}

	cmd := exec.Command("sh", "-c", command)
	if job.cwd != "" {
		cmd.Dir = job.cwd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", 0, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", 0, err
	}
	if err := cmd.Start(); err != nil {
		return "", 0, err
	}
	job.pid = cmd.Process.Pid
	if pgid, err := syscall.Getpgid(job.pid); err == nil {
		job.pgid = pgid
	}

	m.session(sessionID).add(job)

	go job.pump(stdout)
	go job.pump(stderr)
	go func() {
		err := cmd.Wait()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				job.exitCode = exit.ExitCode()
			} else {
				job.exitCode = -1
			}
		}
		job.running = false
		close(job.done)
	}()

	return id, job.pid, nil
}

func (j *backgroundJob) pump(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			j.buffer.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (m *JobManager) Logs(sessionID, jobID string, lines int) (string, error) {
	job, err := m.job(sessionID, jobID)
	if err != nil {
		return "", err
	}
	if lines <= 0 {
		lines = 50
	}
	status := "running"
	if !job.running {
		status = fmt.Sprintf("exited (code %d)", job.exitCode)
	}
	logText := job.buffer.LastLines(lines)
	return fmt.Sprintf("job_id=%s pid=%d status=%s\ncommand: %s\n---\n%s",
		job.id, job.pid, status, job.command, logText), nil
}

func (m *JobManager) Kill(sessionID, jobID string) (string, error) {
	job, err := m.job(sessionID, jobID)
	if err != nil {
		return "", err
	}
	if job.running {
		m.killProcessGroup(job)
		job.running = false
	}
	return fmt.Sprintf("killed job %s (pid %d)", job.id, job.pid), nil
}

func (m *JobManager) KillSession(sessionID string) {
	if sessionID == "" {
		return
	}
	m.mu.Lock()
	sj := m.sessions[sessionID]
	m.mu.Unlock()
	if sj == nil {
		return
	}
	sj.mu.Lock()
	for _, job := range sj.jobs {
		if job.running {
			m.killProcessGroup(job)
			job.running = false
		}
	}
	sj.mu.Unlock()
}

func (m *JobManager) KillAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = map[string]*sessionJobs{}
	m.mu.Unlock()
	for _, sj := range sessions {
		sj.mu.Lock()
		for _, job := range sj.jobs {
			if job.running {
				m.killProcessGroup(job)
				job.running = false
			}
		}
		sj.mu.Unlock()
	}
}

func (m *JobManager) killProcessGroup(job *backgroundJob) {
	if job.pgid > 0 {
		_ = syscall.Kill(-job.pgid, syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)
		_ = syscall.Kill(-job.pgid, syscall.SIGKILL)
		return
	}
	if job.pid > 0 {
		_ = syscall.Kill(job.pid, syscall.SIGKILL)
	}
}

func (m *JobManager) session(id string) *sessionJobs {
	m.mu.Lock()
	defer m.mu.Unlock()
	sj, ok := m.sessions[id]
	if !ok {
		sj = &sessionJobs{jobs: map[string]*backgroundJob{}}
		m.sessions[id] = sj
	}
	return sj
}

func (sj *sessionJobs) add(job *backgroundJob) {
	sj.mu.Lock()
	defer sj.mu.Unlock()
	sj.jobs[job.id] = job
}

func (m *JobManager) job(sessionID, jobID string) (*backgroundJob, error) {
	if sessionID == "" {
		sessionID = "default"
	}
	m.mu.Lock()
	sj := m.sessions[sessionID]
	m.mu.Unlock()
	if sj == nil {
		return nil, fmt.Errorf("unknown job %q", jobID)
	}
	sj.mu.Lock()
	job := sj.jobs[jobID]
	sj.mu.Unlock()
	if job == nil {
		return nil, fmt.Errorf("unknown job %q", jobID)
	}
	return job, nil
}

type lineRingBuffer struct {
	mu       sync.Mutex
	maxBytes int
	maxLines int
	lines    []string
	bytes    int
}

func newLineRingBuffer(maxBytes, maxLines int) *lineRingBuffer {
	return &lineRingBuffer{maxBytes: maxBytes, maxLines: maxLines}
}

func (b *lineRingBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	chunk := string(p)
	parts := strings.Split(chunk, "\n")
	if len(b.lines) > 0 && len(parts) > 0 {
		b.lines[len(b.lines)-1] += parts[0]
		parts = parts[1:]
	}
	for _, part := range parts {
		b.lines = append(b.lines, part)
		b.bytes += len(part) + 1
	}
	for len(b.lines) > b.maxLines || b.bytes > b.maxBytes {
		if len(b.lines) == 0 {
			break
		}
		removed := b.lines[0]
		b.lines = b.lines[1:]
		b.bytes -= len(removed) + 1
	}
}

func (b *lineRingBuffer) LastLines(n int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.lines) == 0 {
		return "(no output yet)"
	}
	start := 0
	if len(b.lines) > n {
		start = len(b.lines) - n
	}
	return strings.Join(b.lines[start:], "\n")
}
