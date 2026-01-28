package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	// ErrHookTimeout is returned when the hook execution times out.
	ErrHookTimeout = errors.New("hook execution timed out")
	// ErrHookFailed is returned when the hook command fails.
	ErrHookFailed = errors.New("hook command failed")
)

// HookExecutor handles execution of backup hooks inside Docker containers.
type HookExecutor struct {
	dockerBinary string
	logger       zerolog.Logger
}

// NewHookExecutor creates a new HookExecutor.
func NewHookExecutor(logger zerolog.Logger) *HookExecutor {
	return &HookExecutor{
		dockerBinary: "docker",
		logger:       logger.With().Str("component", "docker_hooks").Logger(),
	}
}

// NewHookExecutorWithBinary creates a new HookExecutor with a custom docker binary path.
func NewHookExecutorWithBinary(binary string, logger zerolog.Logger) *HookExecutor {
	return &HookExecutor{
		dockerBinary: binary,
		logger:       logger.With().Str("component", "docker_hooks").Logger(),
	}
}

// ExecuteHook runs a hook command inside the specified container.
func (e *HookExecutor) ExecuteHook(ctx context.Context, hook *models.ContainerBackupHook, backupID uuid.UUID) (*models.ContainerHookExecution, error) {
	e.logger.Info().
		Str("hook_id", hook.ID.String()).
		Str("container", hook.ContainerName).
		Str("type", string(hook.Type)).
		Int("timeout_seconds", hook.TimeoutSeconds).
		Msg("executing container hook")

	// Get the command to execute (from template or custom)
	command := e.getCommand(hook)
	if command == "" {
		return nil, errors.New("no command specified for hook")
	}

	// Create execution record
	execution := &models.ContainerHookExecution{
		HookID:    hook.ID,
		BackupID:  backupID,
		Container: hook.ContainerName,
		Type:      hook.Type,
		Command:   command,
		StartedAt: time.Now(),
	}

	// Create context with timeout
	hookCtx, cancel := context.WithTimeout(ctx, time.Duration(hook.TimeoutSeconds)*time.Second)
	defer cancel()

	// Build docker exec command
	args := e.buildExecArgs(hook, command)

	e.logger.Debug().
		Strs("args", args).
		Msg("running docker exec")

	// Execute the command
	cmd := exec.CommandContext(hookCtx, e.dockerBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execution.CompletedAt = time.Now()
	execution.Duration = execution.CompletedAt.Sub(execution.StartedAt)

	// Combine stdout and stderr for output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate output if too long (max 64KB)
	const maxOutputLen = 64 * 1024
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen] + "\n... (output truncated)"
	}
	execution.Output = output

	// Handle errors
	if err != nil {
		if hookCtx.Err() == context.DeadlineExceeded {
			execution.Error = fmt.Sprintf("hook timed out after %d seconds", hook.TimeoutSeconds)
			execution.ExitCode = -1
			e.logger.Error().
				Str("hook_id", hook.ID.String()).
				Str("container", hook.ContainerName).
				Int("timeout", hook.TimeoutSeconds).
				Msg("hook execution timed out")
			return execution, ErrHookTimeout
		}

		// Get exit code if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			execution.ExitCode = exitErr.ExitCode()
		} else {
			execution.ExitCode = -1
		}
		execution.Error = err.Error()

		// Check for container not found
		if strings.Contains(output, "No such container") || strings.Contains(err.Error(), "No such container") {
			e.logger.Error().
				Str("hook_id", hook.ID.String()).
				Str("container", hook.ContainerName).
				Msg("container not found")
			return execution, ErrContainerNotFound
		}

		e.logger.Error().
			Err(err).
			Str("hook_id", hook.ID.String()).
			Str("container", hook.ContainerName).
			Int("exit_code", execution.ExitCode).
			Str("output", output).
			Msg("hook execution failed")

		return execution, fmt.Errorf("%w: exit code %d", ErrHookFailed, execution.ExitCode)
	}

	execution.ExitCode = 0

	e.logger.Info().
		Str("hook_id", hook.ID.String()).
		Str("container", hook.ContainerName).
		Dur("duration", execution.Duration).
		Msg("hook execution completed successfully")

	return execution, nil
}

// buildExecArgs builds the docker exec command arguments.
func (e *HookExecutor) buildExecArgs(hook *models.ContainerBackupHook, command string) []string {
	args := []string{"exec"}

	// Add working directory if specified
	if hook.WorkingDir != "" {
		args = append(args, "-w", hook.WorkingDir)
	}

	// Add user if specified
	if hook.User != "" {
		args = append(args, "-u", hook.User)
	}

	// Add container name
	args = append(args, hook.ContainerName)

	// Add the command (using sh -c for shell interpretation)
	args = append(args, "sh", "-c", command)

	return args
}

// getCommand returns the command to execute, either from template or custom.
func (e *HookExecutor) getCommand(hook *models.ContainerBackupHook) string {
	if hook.Template == models.ContainerHookTemplateNone || hook.Template == "" {
		return hook.Command
	}

	// Get template command
	templateCmd := GetTemplateCommand(hook.Template, hook.Type, hook.TemplateVars)
	if templateCmd != "" {
		return templateCmd
	}

	// Fall back to custom command if template not found
	return hook.Command
}

// ExecutePreBackupHooks runs all pre-backup hooks for a schedule.
func (e *HookExecutor) ExecutePreBackupHooks(ctx context.Context, hooks []*models.ContainerBackupHook, backupID uuid.UUID) ([]*models.ContainerHookExecution, error) {
	return e.executeHooks(ctx, hooks, models.ContainerHookTypePreBackup, backupID)
}

// ExecutePostBackupHooks runs all post-backup hooks for a schedule.
func (e *HookExecutor) ExecutePostBackupHooks(ctx context.Context, hooks []*models.ContainerBackupHook, backupID uuid.UUID) ([]*models.ContainerHookExecution, error) {
	return e.executeHooks(ctx, hooks, models.ContainerHookTypePostBackup, backupID)
}

// executeHooks runs all hooks of the specified type.
func (e *HookExecutor) executeHooks(ctx context.Context, hooks []*models.ContainerBackupHook, hookType models.ContainerHookType, backupID uuid.UUID) ([]*models.ContainerHookExecution, error) {
	var executions []*models.ContainerHookExecution
	var firstError error

	for _, hook := range hooks {
		// Skip disabled hooks
		if !hook.Enabled {
			continue
		}

		// Skip hooks of different type
		if hook.Type != hookType {
			continue
		}

		execution, err := e.ExecuteHook(ctx, hook, backupID)
		if execution != nil {
			executions = append(executions, execution)
		}

		if err != nil {
			if hook.FailOnError {
				// Return immediately if this hook should fail the backup
				if firstError == nil {
					firstError = err
				}
				return executions, fmt.Errorf("hook %s failed: %w", hook.ID.String(), err)
			}
			// Log but continue for non-critical hooks
			e.logger.Warn().
				Err(err).
				Str("hook_id", hook.ID.String()).
				Str("container", hook.ContainerName).
				Msg("hook failed but fail_on_error is false, continuing")
		}
	}

	return executions, nil
}

