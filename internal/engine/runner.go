package engine

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/internal/interfaces"
	"github.com/themobileprof/clipilot/internal/models"
	"github.com/themobileprof/clipilot/internal/utils/safeexec"
)

// Runner executes module flows
type Runner struct {
	db      *sql.DB
	loader  interfaces.ModuleStore
	dryRun  bool
	autoYes bool
}

// NewRunner creates a new flow runner
func NewRunner(db *sql.DB, loader interfaces.ModuleStore) *Runner {
	return &Runner{
		db:     db,
		loader: loader,
	}
}

// Ensure Runner implements FlowRunner interface
var _ interfaces.FlowRunner = (*Runner)(nil)

// SetDryRun enables/disables dry-run mode
func (r *Runner) SetDryRun(enabled bool) {
	r.dryRun = enabled
}

// SetAutoYes enables/disables auto-confirmation
func (r *Runner) SetAutoYes(enabled bool) {
	r.autoYes = enabled
}

// Run executes a module flow
func (r *Runner) Run(moduleID string) error {
	// Load module
	module, err := r.loader.GetModule(moduleID)
	if err != nil {
		return fmt.Errorf("failed to load module: %w", err)
	}

	// Create execution context
	ctx := &models.ExecutionContext{
		SessionID:   fmt.Sprintf("session_%d", time.Now().Unix()),
		ModuleID:    moduleID,
		FlowName:    "main",
		CurrentStep: "",
		State:       make(map[string]string),
		DryRun:      r.dryRun,
	}

	// Log execution start
	logID, err := r.logStart(ctx, moduleID)
	if err != nil {
		return fmt.Errorf("failed to log start: %w", err)
	}
	ctx.LogID = logID

	startTime := time.Now()
	var execErr error

	// Run flow
	flow, exists := module.Flows[ctx.FlowName]
	if !exists {
		execErr = fmt.Errorf("flow 'main' not found in module")
	} else {
		fmt.Printf("\n=== Running Module: %s ===\n", module.Name)
		fmt.Printf("Description: %s\n", module.Description)
		if r.dryRun {
			fmt.Println("[DRY RUN MODE - Commands will not be executed]")
		}
		fmt.Println()

		execErr = r.runFlow(ctx, module, flow)
	}

	// Log completion
	duration := time.Since(startTime).Milliseconds()
	status := "completed"
	errorMsg := ""
	if execErr != nil {
		status = "failed"
		errorMsg = execErr.Error()
	}

	if err := r.logComplete(ctx.LogID, status, errorMsg, duration); err != nil {
		fmt.Printf("Warning: failed to log completion: %v\n", err)
	}

	return execErr
}

// runFlow executes a flow's steps
func (r *Runner) runFlow(ctx *models.ExecutionContext, module *models.Module, flow *models.Flow) error {
	currentStepKey := flow.Start
	stepCount := 0
	maxSteps := 100 // Prevent infinite loops

	for currentStepKey != "" && stepCount < maxSteps {
		step, exists := flow.Steps[currentStepKey]
		if !exists {
			return fmt.Errorf("step not found: %s", currentStepKey)
		}

		stepCount++
		ctx.CurrentStep = currentStepKey

		// Check condition if present
		if step.Condition != nil {
			if !r.evaluateCondition(ctx, step.Condition) {
				currentStepKey = step.Next
				continue
			}
		}

		// Execute step based on type
		nextStep, err := r.executeStep(ctx, module, step)
		if err != nil {
			return fmt.Errorf("step %s failed: %w", currentStepKey, err)
		}

		currentStepKey = nextStep
	}

	if stepCount >= maxSteps {
		return fmt.Errorf("maximum step count exceeded (possible infinite loop)")
	}

	return nil
}

// executeStep executes a single step
func (r *Runner) executeStep(ctx *models.ExecutionContext, module *models.Module, step *models.Step) (string, error) {
	switch step.Type {
	case "action":
		return r.executeAction(ctx, step)
	case "instruction":
		return r.executeInstruction(ctx, step)
	case "branch":
		return r.executeBranch(ctx, step)
	case "terminal":
		return r.executeTerminal(step)
	default:
		return "", fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// executeAction runs a sub-module
func (r *Runner) executeAction(ctx *models.ExecutionContext, step *models.Step) (string, error) {
	if step.RunModule == "" {
		return step.Next, nil
	}

	fmt.Printf("[Action] Running module: %s\n", step.RunModule)

	// Create a new runner for the sub-module
	subRunner := NewRunner(r.db, r.loader)
	subRunner.SetDryRun(r.dryRun)
	subRunner.SetAutoYes(r.autoYes)

	if err := subRunner.Run(step.RunModule); err != nil {
		return "", fmt.Errorf("sub-module failed: %w", err)
	}

	return step.Next, nil
}

// executeInstruction shows a message and optionally runs a command
func (r *Runner) executeInstruction(ctx *models.ExecutionContext, step *models.Step) (string, error) {
	if step.Message != "" {
		fmt.Printf("\n[Step] %s\n", step.Message)
	}

	if step.Command != "" {
		fmt.Printf("  Command: %s\n", step.Command)

		if r.dryRun {
			fmt.Println("  [Dry run - command not executed]")
		} else {
			// Ask for confirmation
			if !r.autoYes {
				confirmed := r.confirm("  Run this command?")
				if !confirmed {
					fmt.Println("  [Skipped by user]")
					return step.Next, nil
				}
			}

			// Execute command
			output, err := r.runCommand(step.Command)
			if err != nil {
				fmt.Printf("  ✗ Command failed: %v\n", err)
				return "", err
			}

			if output != "" {
				fmt.Printf("  Output: %s\n", strings.TrimSpace(output))
			}
			fmt.Println("  ✓ Success")

			// Run validations
			if len(step.Validate) > 0 {
				if err := r.runValidations(ctx, step); err != nil {
					return "", fmt.Errorf("validation failed: %w", err)
				}
			}
		}
	}

	return step.Next, nil
}

// executeBranch evaluates a branch and selects next step
func (r *Runner) executeBranch(ctx *models.ExecutionContext, step *models.Step) (string, error) {
	if step.BasedOn == "" {
		return "", fmt.Errorf("branch step missing 'based_on' field")
	}

	value, exists := ctx.State[step.BasedOn]
	if !exists {
		return "", fmt.Errorf("state key not found: %s", step.BasedOn)
	}

	nextStep, exists := step.Map[value]
	if !exists {
		return "", fmt.Errorf("no branch mapping for value: %s", value)
	}

	fmt.Printf("[Branch] %s = %s → %s\n", step.BasedOn, value, nextStep)
	return nextStep, nil
}

// executeTerminal displays final message
func (r *Runner) executeTerminal(step *models.Step) (string, error) {
	if step.Message != "" {
		fmt.Printf("\n✓ %s\n", step.Message)
	}
	return "", nil // Empty string signals flow end
}

// evaluateCondition checks if a condition is met
func (r *Runner) evaluateCondition(ctx *models.ExecutionContext, cond *models.Condition) bool {
	value, exists := ctx.State[cond.StateKey]
	if !exists {
		return false
	}

	switch cond.Operator {
	case "eq":
		return value == cond.Value
	case "ne":
		return value != cond.Value
	case "contains":
		return strings.Contains(value, cond.Value)
	default:
		return false
	}
}

// runCommand executes a shell command
func (r *Runner) runCommand(command string) (string, error) {
	cmd := safeexec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runValidations runs validation checks
func (r *Runner) runValidations(ctx *models.ExecutionContext, step *models.Step) error {
	for _, validation := range step.Validate {
		if validation.CheckCommand != "" {
			output, err := r.runCommand(validation.CheckCommand)
			if err != nil {
				msg := validation.ErrorMessage
				if msg == "" {
					msg = fmt.Sprintf("validation command failed: %s", validation.CheckCommand)
				}
				return fmt.Errorf("%s: %w", msg, err)
			}

			// Check expected output if specified
			if validation.Expected != "" && !strings.Contains(output, validation.Expected) {
				msg := validation.ErrorMessage
				if msg == "" {
					msg = fmt.Sprintf("output does not contain expected value: %s", validation.Expected)
				}
				return fmt.Errorf("%s", msg)
			}

			fmt.Println("  ✓ Validation passed")
		}
	}
	return nil
}

// confirm prompts user for yes/no confirmation
func (r *Runner) confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// logStart logs the beginning of execution
func (r *Runner) logStart(ctx *models.ExecutionContext, moduleID string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO logs (session_id, input, resolved_module, confidence, method, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ctx.SessionID, "", moduleID, 1.0, "direct", "started")
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// logComplete logs the completion of execution
func (r *Runner) logComplete(logID int64, status, errorMsg string, durationMs int64) error {
	_, err := r.db.Exec(`
		UPDATE logs SET status = ?, error_message = ?, duration_ms = ?
		WHERE id = ?
	`, status, errorMsg, durationMs, logID)
	return err
}
