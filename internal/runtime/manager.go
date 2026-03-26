package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bssm-oss/PlainCode/internal/execenv"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

const defaultHealthTimeout = 15 * time.Second

// StartOptions controls how a managed runtime is started.
type StartOptions struct {
	Mode          string
	HealthTimeout time.Duration
}

// Manager starts, stops, and inspects spec-managed services.
type Manager struct {
	projectDir string
	store      *Store
}

// NewManager creates a runtime manager rooted at a project directory.
func NewManager(projectDir, stateDir string) *Manager {
	return &Manager{
		projectDir: projectDir,
		store:      NewStore(stateDir),
	}
}

func (m *Manager) recordSpecEvent(specID, mode, kind, message string, fields map[string]string) {
	_ = m.store.AppendEvent(specID, Event{
		Timestamp: time.Now(),
		Kind:      kind,
		Message:   message,
		Fields:    withBaseFields(mode, fields),
	})
}

func (m *Manager) recordStateEvent(state *State, kind, message string, fields map[string]string) {
	state.LastEvent = kind
	base := map[string]string{
		"status": state.Status,
		"health": state.Health,
	}
	if state.Mode == ModeProcess {
		if state.PID != 0 {
			base["pid"] = fmt.Sprintf("%d", state.PID)
		}
		if len(state.Command) > 0 {
			base["command"] = strings.Join(state.Command, " ")
		}
	}
	if state.Mode == ModeDocker {
		if state.ContainerName != "" {
			base["container_name"] = state.ContainerName
		}
		if state.ContainerID != "" {
			base["container_id"] = state.ContainerID
		}
		if state.Image != "" {
			base["image"] = state.Image
		}
	}
	for key, value := range fields {
		base[key] = value
	}
	_ = m.store.AppendEvent(state.SpecID, Event{
		Timestamp: time.Now(),
		Kind:      kind,
		Message:   message,
		Fields:    withBaseFields(state.Mode, base),
	})
}

// Start launches the runtime declared by a spec.
func (m *Manager) Start(ctx context.Context, spec *ast.Spec, opts StartOptions) (*State, error) {
	mode, err := m.resolveMode(spec, opts.Mode)
	if err != nil {
		return nil, err
	}
	m.recordSpecEvent(spec.ID, mode, "start_requested", "runtime start requested", nil)

	if existing, err := m.refresh(spec.ID); err == nil && existing.Status == StatusRunning {
		m.recordStateEvent(existing, "start_rejected", "runtime is already running", nil)
		return nil, fmt.Errorf("spec %s is already running via %s", spec.ID, existing.Mode)
	}

	timeout := opts.HealthTimeout
	if timeout <= 0 {
		timeout = defaultHealthTimeout
	}

	switch mode {
	case ModeProcess:
		return m.startProcess(spec, timeout)
	case ModeDocker:
		return m.startDocker(ctx, spec, timeout)
	default:
		return nil, fmt.Errorf("unsupported runtime mode: %s", mode)
	}
}

// Stop terminates the runtime declared for a spec.
func (m *Manager) Stop(ctx context.Context, specID string) (*State, error) {
	state, err := m.store.Load(specID)
	if err != nil {
		return nil, err
	}
	m.recordStateEvent(state, "stop_requested", "runtime stop requested", nil)

	switch state.Mode {
	case ModeProcess:
		err = stopProcess(state)
	case ModeDocker:
		_ = m.captureDockerLogs(ctx, state)
		err = stopDocker(ctx, m.projectDir, state)
	default:
		err = fmt.Errorf("unknown runtime mode: %s", state.Mode)
	}
	if err != nil {
		state.Error = err.Error()
		m.recordStateEvent(state, "stop_failed", "runtime stop failed", map[string]string{"error": err.Error()})
		return nil, err
	}

	state.Status = StatusStopped
	state.Health = HealthUnknown
	state.Error = ""
	state.LastCheckAt = time.Now()
	state.StoppedAt = time.Now()
	m.recordStateEvent(state, "stop_succeeded", "runtime stopped", nil)
	if err := m.store.Save(state); err != nil {
		return nil, err
	}
	return state, nil
}

// Status returns the refreshed runtime status for a spec ID.
func (m *Manager) Status(ctx context.Context, specID string) (*State, error) {
	state, err := m.store.Load(specID)
	if err != nil {
		return nil, err
	}
	return m.refreshWithContext(ctx, state)
}

// List returns all known runtime states, refreshed from the underlying system.
func (m *Manager) List(ctx context.Context) ([]*State, error) {
	states, err := m.store.ListAll()
	if err != nil {
		return nil, err
	}

	var refreshed []*State
	for _, state := range states {
		next, err := m.refreshWithContext(ctx, state)
		if err != nil {
			state.Status = StatusUnknown
			state.Health = HealthUnknown
			state.Error = err.Error()
			refreshed = append(refreshed, state)
			continue
		}
		refreshed = append(refreshed, next)
	}

	sort.Slice(refreshed, func(i, j int) bool {
		return refreshed[i].SpecID < refreshed[j].SpecID
	})
	return refreshed, nil
}

func (m *Manager) startProcess(spec *ast.Spec, timeout time.Duration) (*State, error) {
	command := strings.Fields(strings.TrimSpace(spec.Runtime.Process.Command))
	if len(command) == 0 {
		command = inferredProcessCommand(spec)
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("spec %s does not declare runtime.process.command and no default process command could be inferred", spec.ID)
	}

	workingDir := m.projectDir
	if rel := strings.TrimSpace(spec.Runtime.Process.WorkingDir); rel != "" {
		workingDir = filepath.Join(m.projectDir, rel)
	}

	if err := os.MkdirAll(m.store.dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating runtime state directory: %w", err)
	}
	healthcheckURL := healthcheckURLFromSpec(spec.Runtime.HealthURL, spec.Runtime.Process.HealthcheckURL)
	logPath := m.store.LogPath(spec.ID)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening runtime log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(execenv.ResolveBinary(command[0]), command[1:]...)
	cmd.Dir = workingDir
	cmd.Env = mergeEnv(os.Environ(), spec.Runtime.Process.Env)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		state := &State{
			SpecID:         spec.ID,
			Mode:           ModeProcess,
			Status:         StatusStopped,
			Health:         HealthUnhealthy,
			Command:        command,
			WorkingDir:     workingDir,
			Env:            cloneMap(spec.Runtime.Process.Env),
			HealthcheckURL: healthcheckURL,
			LogPath:        logPath,
			Error:          err.Error(),
			LastCheckAt:    time.Now(),
			StoppedAt:      time.Now(),
		}
		m.recordStateEvent(state, "start_failed", "process runtime failed to start", map[string]string{"error": err.Error()})
		_ = m.store.Save(state)
		return nil, fmt.Errorf("starting process runtime: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	state := &State{
		SpecID:         spec.ID,
		Mode:           ModeProcess,
		Status:         StatusRunning,
		Health:         HealthUnknown,
		Command:        command,
		WorkingDir:     workingDir,
		Env:            cloneMap(spec.Runtime.Process.Env),
		HealthcheckURL: healthcheckURL,
		LogPath:        logPath,
		PID:            pid,
		PGID:           pid,
		StartedAt:      time.Now(),
		LastCheckAt:    time.Now(),
	}
	m.recordStateEvent(state, "process_spawned", "process runtime started", nil)

	if err := waitForProcessReady(state.PID, state.HealthcheckURL, timeout); err != nil {
		_ = stopProcess(state)
		state.Status = StatusStopped
		state.Health = HealthUnhealthy
		state.Error = err.Error()
		state.LastCheckAt = time.Now()
		state.StoppedAt = time.Now()
		m.recordStateEvent(state, "start_failed", "process runtime failed to become healthy", map[string]string{"error": err.Error()})
		_ = m.store.Save(state)
		return nil, fmt.Errorf("process healthcheck failed: %w", err)
	}
	if state.HealthcheckURL != "" {
		state.Health = HealthHealthy
	}
	state.LastCheckAt = time.Now()
	m.recordStateEvent(state, "start_succeeded", "process runtime is ready", nil)

	if err := m.store.Save(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (m *Manager) startDocker(ctx context.Context, spec *ast.Spec, timeout time.Duration) (*State, error) {
	cfg := spec.Runtime.Docker
	contextPath := strings.TrimSpace(cfg.Context)
	if contextPath == "" {
		contextPath = "."
	}
	dockerfile := strings.TrimSpace(cfg.Dockerfile)
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	image := strings.TrimSpace(cfg.Image)
	if image == "" {
		image = defaultImageName(spec.ID)
	}
	containerName := strings.TrimSpace(cfg.ContainerName)
	if containerName == "" {
		containerName = defaultContainerName(spec.ID)
	}
	logPath := m.store.LogPath(spec.ID)

	_ = removeDockerContainer(ctx, m.projectDir, containerName)
	m.recordSpecEvent(spec.ID, ModeDocker, "docker_build_started", "docker build started", map[string]string{
		"image":   image,
		"command": strings.Join([]string{"docker", "build", "-t", image, "-f", dockerfile, contextPath}, " "),
	})

	buildArgs := []string{"build", "-t", image, "-f", dockerfile, contextPath}
	buildOutput, err := execInDirCapture(ctx, m.projectDir, "docker", buildArgs...)
	if strings.TrimSpace(buildOutput) != "" {
		_ = m.store.WriteLog(spec.ID, []byte(buildOutput))
	}
	if err != nil {
		m.recordSpecEvent(spec.ID, ModeDocker, "docker_build_failed", "docker build failed", map[string]string{
			"image":          image,
			"error":          err.Error(),
			"output_snippet": snippet(buildOutput),
		})
		return nil, fmt.Errorf("docker build failed: %w", err)
	}
	m.recordSpecEvent(spec.ID, ModeDocker, "docker_build_succeeded", "docker build completed", map[string]string{
		"image": image,
	})

	runArgs := []string{"run", "-d", "--name", containerName}
	for _, port := range cfg.Ports {
		if strings.TrimSpace(port) == "" {
			continue
		}
		runArgs = append(runArgs, "-p", port)
	}
	for _, entry := range mapToEnvList(cfg.Env) {
		runArgs = append(runArgs, "-e", entry)
	}
	runArgs = append(runArgs, image)
	m.recordSpecEvent(spec.ID, ModeDocker, "docker_run_started", "docker run started", map[string]string{
		"container_name": containerName,
		"command":        strings.Join(append([]string{"docker"}, runArgs...), " "),
	})

	containerID, err := execInDirOutput(ctx, m.projectDir, "docker", runArgs...)
	if err != nil {
		m.recordSpecEvent(spec.ID, ModeDocker, "docker_run_failed", "docker run failed", map[string]string{
			"container_name": containerName,
			"error":          err.Error(),
		})
		return nil, fmt.Errorf("docker run failed: %w", err)
	}

	state := &State{
		SpecID:         spec.ID,
		Mode:           ModeDocker,
		Status:         StatusRunning,
		Health:         HealthUnknown,
		Env:            cloneMap(cfg.Env),
		HealthcheckURL: healthcheckURLFromSpec(spec.Runtime.HealthURL, cfg.HealthcheckURL),
		Image:          image,
		ContainerName:  containerName,
		ContainerID:    strings.TrimSpace(containerID),
		LogPath:        logPath,
		Ports:          append([]string(nil), cfg.Ports...),
		StartedAt:      time.Now(),
		LastCheckAt:    time.Now(),
	}
	m.recordStateEvent(state, "docker_run_succeeded", "docker container started", nil)

	if err := waitForDockerReady(ctx, m.projectDir, state, timeout); err != nil {
		_ = m.captureDockerLogs(ctx, state)
		_ = stopDocker(ctx, m.projectDir, state)
		state.Status = StatusStopped
		state.Health = HealthUnhealthy
		state.Error = err.Error()
		state.LastCheckAt = time.Now()
		state.StoppedAt = time.Now()
		m.recordStateEvent(state, "start_failed", "docker runtime failed to become healthy", map[string]string{"error": err.Error()})
		_ = m.store.Save(state)
		return nil, fmt.Errorf("docker healthcheck failed: %w", err)
	}
	if state.HealthcheckURL != "" {
		state.Health = HealthHealthy
	}
	state.LastCheckAt = time.Now()
	_ = m.captureDockerLogs(ctx, state)
	m.recordStateEvent(state, "start_succeeded", "docker runtime is ready", nil)

	if err := m.store.Save(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (m *Manager) resolveMode(spec *ast.Spec, requested string) (string, error) {
	mode := strings.TrimSpace(requested)
	if mode == "" {
		mode = runtimeModeFromSpec(spec)
	}
	if mode == "" {
		mode = ModeAuto
	}
	if mode == ModeAuto {
		if hasDockerConfig(spec) {
			mode = ModeDocker
		} else if hasProcessConfig(spec) {
			mode = ModeProcess
		} else if fileExists(filepath.Join(m.projectDir, "Dockerfile")) {
			mode = ModeDocker
		} else if len(inferredProcessCommand(spec)) > 0 {
			mode = ModeProcess
		}
	}

	switch mode {
	case ModeProcess, ModeDocker:
		return mode, nil
	case "":
		return "", fmt.Errorf("spec %s does not define a runtime mode; add runtime.process or runtime.docker", spec.ID)
	default:
		return "", fmt.Errorf("invalid runtime mode %q", mode)
	}
}

func (m *Manager) refresh(specID string) (*State, error) {
	return m.Status(context.Background(), specID)
}

func (m *Manager) refreshWithContext(ctx context.Context, state *State) (*State, error) {
	prevStatus := state.Status
	prevHealth := state.Health
	prevError := state.Error
	if state.Status == StatusStopped {
		state.Health = HealthUnknown
		state.Error = ""
		state.LastCheckAt = time.Now()
		return state, nil
	}

	switch state.Mode {
	case ModeProcess:
		if processExists(state.PID) {
			state.Status = StatusRunning
		} else {
			state.Status = StatusStopped
			if state.StoppedAt.IsZero() {
				state.StoppedAt = time.Now()
			}
		}
	case ModeDocker:
		running, err := dockerRunning(ctx, m.projectDir, state.ContainerName)
		if err != nil {
			state.Status = StatusUnknown
			state.Health = HealthUnknown
			state.Error = err.Error()
			_ = m.store.Save(state)
			return state, nil
		}
		if running {
			state.Status = StatusRunning
		} else {
			state.Status = StatusStopped
			if state.StoppedAt.IsZero() {
				state.StoppedAt = time.Now()
			}
		}
	default:
		return nil, fmt.Errorf("unknown runtime mode: %s", state.Mode)
	}

	state.Error = ""
	state.Health = HealthUnknown
	if state.Status == StatusRunning && state.HealthcheckURL != "" {
		if err := waitForHealthcheck(state.HealthcheckURL, 2*time.Second); err != nil {
			state.Health = HealthUnhealthy
			state.Error = err.Error()
		} else {
			state.Health = HealthHealthy
		}
	}
	state.LastCheckAt = time.Now()
	if state.Mode == ModeDocker {
		_ = m.captureDockerLogs(ctx, state)
	}
	if state.Status != prevStatus || state.Health != prevHealth || state.Error != prevError {
		message := fmt.Sprintf("runtime state changed from %s/%s to %s/%s", prevStatus, prevHealth, state.Status, state.Health)
		fields := map[string]string{
			"previous_status": prevStatus,
			"previous_health": prevHealth,
		}
		if prevError != "" {
			fields["previous_error"] = prevError
		}
		if state.Error != "" {
			fields["error"] = state.Error
		}
		m.recordStateEvent(state, "state_changed", message, fields)
	}

	if err := m.store.Save(state); err != nil {
		return nil, err
	}
	return state, nil
}

func hasProcessConfig(spec *ast.Spec) bool {
	cfg := spec.Runtime.Process
	return strings.TrimSpace(cfg.Command) != "" || strings.TrimSpace(cfg.WorkingDir) != "" || len(cfg.Env) > 0 || strings.TrimSpace(healthcheckURLFromSpec(spec.Runtime.HealthURL, cfg.HealthcheckURL)) != ""
}

func hasDockerConfig(spec *ast.Spec) bool {
	cfg := spec.Runtime.Docker
	return strings.TrimSpace(cfg.Context) != "" || strings.TrimSpace(cfg.Dockerfile) != "" || strings.TrimSpace(cfg.Image) != "" || strings.TrimSpace(cfg.ContainerName) != "" || len(cfg.Ports) > 0 || len(cfg.Env) > 0 || strings.TrimSpace(healthcheckURLFromSpec(spec.Runtime.HealthURL, cfg.HealthcheckURL)) != ""
}

func runtimeModeFromSpec(spec *ast.Spec) string {
	mode := strings.TrimSpace(spec.Runtime.DefaultMode)
	if mode != "" {
		return mode
	}
	return strings.TrimSpace(spec.Runtime.Mode)
}

func healthcheckURLFromSpec(legacy, specific string) string {
	if strings.TrimSpace(specific) != "" {
		return strings.TrimSpace(specific)
	}
	return strings.TrimSpace(legacy)
}

func inferredProcessCommand(spec *ast.Spec) []string {
	switch strings.ToLower(strings.TrimSpace(spec.Language)) {
	case "go":
		return []string{"go", "run", "."}
	default:
		return nil
	}
}

func waitForHealthcheck(rawURL string, timeout time.Duration) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = defaultHealthTimeout
	}

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(rawURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return nil
			}
			lastErr = fmt.Errorf("healthcheck returned %s", resp.Status)
		} else {
			lastErr = err
		}
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("healthcheck timed out")
	}
	return lastErr
}

func waitForProcessReady(pid int, rawURL string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultHealthTimeout
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return fmt.Errorf("process exited before becoming ready")
		}
		if strings.TrimSpace(rawURL) == "" {
			time.Sleep(300 * time.Millisecond)
			if processExists(pid) {
				return nil
			}
			return fmt.Errorf("process exited before becoming ready")
		}
		if err := waitForHealthcheck(rawURL, 1*time.Second); err == nil {
			return nil
		}
	}
	return fmt.Errorf("process did not become ready within %s", timeout)
}

func waitForDockerReady(ctx context.Context, dir string, state *State, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultHealthTimeout
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := dockerRunning(ctx, dir, state.ContainerName)
		if err != nil {
			return err
		}
		if !running {
			return fmt.Errorf("container exited before becoming ready")
		}
		if strings.TrimSpace(state.HealthcheckURL) == "" {
			time.Sleep(300 * time.Millisecond)
			running, err := dockerRunning(ctx, dir, state.ContainerName)
			if err != nil {
				return err
			}
			if running {
				return nil
			}
			return fmt.Errorf("container exited before becoming ready")
		}
		if err := waitForHealthcheck(state.HealthcheckURL, 1*time.Second); err == nil {
			return nil
		}
	}
	return fmt.Errorf("container did not become ready within %s", timeout)
}

func (m *Manager) captureDockerLogs(ctx context.Context, state *State) error {
	if state == nil || strings.TrimSpace(state.ContainerName) == "" {
		return nil
	}
	output, err := execInDirCapture(ctx, m.projectDir, "docker", "logs", "--timestamps", state.ContainerName)
	if strings.TrimSpace(output) == "" && err != nil {
		return err
	}
	if strings.TrimSpace(output) == "" {
		return nil
	}
	return m.store.WriteLog(state.SpecID, []byte(output))
}

func stopProcess(state *State) error {
	if state.PID == 0 {
		return nil
	}
	pgid := state.PGID
	if pgid == 0 {
		pgid = state.PID
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
		return fmt.Errorf("sending SIGTERM: %w", err)
	}
	_ = syscall.Kill(state.PID, syscall.SIGTERM)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(state.PID) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
		return fmt.Errorf("sending SIGKILL: %w", err)
	}
	_ = syscall.Kill(state.PID, syscall.SIGKILL)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(state.PID) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("process %d did not exit after SIGKILL", state.PID)
}

func stopDocker(ctx context.Context, dir string, state *State) error {
	if strings.TrimSpace(state.ContainerName) == "" {
		return nil
	}
	if err := execInDir(ctx, dir, "docker", "stop", state.ContainerName); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil
		}
		return fmt.Errorf("stopping docker container: %w", err)
	}
	return nil
}

func dockerRunning(ctx context.Context, dir, containerName string) (bool, error) {
	output, err := execInDirOutput(ctx, dir, "docker", "inspect", "--format", "{{.State.Running}}", containerName)
	if err != nil {
		if strings.Contains(err.Error(), "No such object") || strings.Contains(err.Error(), "No such container") {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(output) == "true", nil
}

func removeDockerContainer(ctx context.Context, dir, containerName string) error {
	if strings.TrimSpace(containerName) == "" {
		return nil
	}
	if err := execInDir(ctx, dir, "docker", "rm", "-f", containerName); err != nil {
		if strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "No such object") {
			return nil
		}
		return err
	}
	return nil
}

func execInDirCapture(ctx context.Context, dir, binary string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, execenv.ResolveBinary(binary), args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	for i, entry := range cmd.Env {
		if strings.HasPrefix(entry, "PATH=") {
			cmd.Env[i] = "PATH=" + execenv.EnsurePath(strings.TrimPrefix(entry, "PATH="))
			goto run
		}
	}
	cmd.Env = append(cmd.Env, "PATH="+execenv.EnsurePath(""))

run:
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %s: %w", binary, strings.Join(args, " "), err)
	}
	return string(output), nil
}

func execInDir(ctx context.Context, dir, binary string, args ...string) error {
	output, err := execInDirCapture(ctx, dir, binary, args...)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(output))
	}
	return nil
}

func execInDirOutput(ctx context.Context, dir, binary string, args ...string) (string, error) {
	output, err := execInDirCapture(ctx, dir, binary, args...)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, strings.TrimSpace(output))
	}
	return strings.TrimSpace(output), nil
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	for _, psPath := range []string{"/bin/ps", "/usr/bin/ps"} {
		if !fileExists(psPath) {
			continue
		}
		ps := exec.Command(psPath, "-o", "stat=", "-p", fmt.Sprintf("%d", pid))
		if output, err := ps.Output(); err == nil {
			stat := strings.TrimSpace(string(output))
			if stat == "" {
				return false
			}
			if strings.HasPrefix(stat, "Z") || strings.Contains(stat, " Z") {
				return false
			}
			return true
		}
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func mergeEnv(base []string, overrides map[string]string) []string {
	env := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	for key, value := range overrides {
		env[key] = value
	}
	env["PATH"] = execenv.EnsurePath(env["PATH"])
	return mapToEnvList(env)
}

func mapToEnvList(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func defaultImageName(specID string) string {
	return "plaincode-" + runtimeSlug(specID)
}

func defaultContainerName(specID string) string {
	return "plaincode-" + runtimeSlug(specID)
}

func runtimeSlug(specID string) string {
	replacer := strings.NewReplacer("/", "-", "_", "-", ":", "-", ".", "-")
	return strings.Trim(replacer.Replace(strings.ToLower(specID)), "-")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func withBaseFields(mode string, fields map[string]string) map[string]string {
	merged := map[string]string{
		"mode": mode,
	}
	for key, value := range fields {
		merged[key] = value
	}
	return merged
}

func snippet(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 400 {
		return text
	}
	return text[:400]
}
