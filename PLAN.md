# Implementation Plan: Add Cron Syntax Support to Scheduler

## Issue
[#29 - Improve schedulers scheduling capabilities](https://github.com/platforma-dev/platforma/issues/29)

Request: Add support for cron syntax in the scheduler package to enable more flexible scheduling patterns.

## Current State Analysis

### Existing Implementation
- **Location**: `scheduler/scheduler.go`
- **Current Behavior**:
  - Scheduler only supports fixed interval scheduling via `time.Duration`
  - Uses `time.Ticker` to execute tasks at regular intervals
  - Takes two parameters: `period time.Duration` and `runner application.Runner`
  - Implements `application.Runner` interface for integration with the application lifecycle

### Usage Patterns
- Demo app: `demo-app/cmd/scheduler/main.go` - Simple example with 1-second interval
- Documentation: `docs/src/content/docs/packages/scheduler.mdx` - Step-by-step guide
- Tests: `scheduler/scheduler_test.go` - Tests for success, error handling, and context cancellation

## Proposed Solution

### Design Approach
**Add cron support while maintaining backward compatibility**

1. Keep existing `New(period, runner)` constructor unchanged
2. Add new `NewWithCron(cronExpr, runner)` constructor for cron-based scheduling
3. Use the `github.com/pardnchiu/go-scheduler` library - modern, feature-rich cron library with:
   - Standard cron syntax (5-field format)
   - Custom descriptors (@hourly, @daily, @weekly, @monthly, @yearly)
   - Interval syntax (@every 5m, @every 2h)
   - Task dependencies, timeouts, and panic recovery
   - Minimal dependencies (stdlib only)
4. Modify internal structure to support both scheduling modes
5. Update `Run()` method to handle both interval and cron schedules

### Technical Implementation

#### 1. Add Cron Library Dependency
```bash
go get github.com/pardnchiu/go-scheduler
```

#### 2. Update Scheduler Structure
```go
type Scheduler struct {
    // Interval-based scheduling
    period time.Duration

    // Cron-based scheduling
    cronExpr string

    // Common fields
    runner application.Runner
    mode   scheduleMode // enum: interval or cron
}

type scheduleMode int

const (
    scheduleModeInterval scheduleMode = iota
    scheduleModeCron
)
```

#### 3. Add New Constructor
```go
// NewWithCron creates a scheduler with cron syntax
// cronExpr examples:
// - "*/5 * * * *" - every 5 minutes
// - "0 */2 * * *" - every 2 hours at minute 0
// - "0 9 * * MON-FRI" - 9 AM on weekdays
func NewWithCron(cronExpr string, runner application.Runner) (*Scheduler, error)
```

#### 4. Update Run Method
- Check `mode` field to determine which scheduling approach to use
- For interval mode: use existing `time.Ticker` logic
- For cron mode: use `github.com/pardnchiu/go-scheduler` internally
- Maintain same logging behavior with trace IDs
- Ensure proper error handling and context cancellation

#### 5. Add Comprehensive Tests
- Test valid cron expressions
- Test invalid cron expressions (should return error from NewWithCron)
- Test cron scheduling execution timing
- Test error handling in cron mode
- Test context cancellation in cron mode
- Ensure existing tests continue to pass (backward compatibility)

#### 6. Update Documentation
Update `docs/src/content/docs/packages/scheduler.mdx`:
- Add section explaining cron syntax support
- Include examples of common cron patterns
- Show side-by-side comparison of interval vs cron approaches
- Add cron expression reference

#### 7. Update Demo Application
Create new demo: `demo-app/cmd/scheduler-cron/main.go`
- Show practical cron usage example
- Demonstrate multiple cron patterns
- Include comments explaining cron syntax

## Implementation Steps

### Phase 1: Core Implementation
1. ✅ Add `github.com/pardnchiu/go-scheduler` to go.mod
2. ✅ Update Scheduler struct with mode field and cronExpr
3. ✅ Implement `NewWithCron()` constructor with validation
4. ✅ Update `Run()` method to handle both modes
5. ✅ Ensure existing `New()` behavior is unchanged

### Phase 2: Testing
6. ✅ Write tests for cron constructor with valid expressions
7. ✅ Write tests for cron constructor with invalid expressions
8. ✅ Write tests for cron execution timing
9. ✅ Write tests for cron error handling and context cancellation
10. ✅ Run existing tests to verify backward compatibility
11. ✅ Run linter: `task lint`

### Phase 3: Documentation & Examples
12. ✅ Update scheduler package documentation
13. ✅ Create new demo app for cron usage
14. ✅ Add examples of common cron patterns

### Phase 4: Validation
15. ✅ Run full test suite: `task test`
16. ✅ Verify test coverage is maintained
17. ✅ Manual testing with demo apps

## Testing Strategy

### Unit Tests (scheduler_test.go)
```go
// Test cases:
- TestNewWithCron_ValidExpression
- TestNewWithCron_InvalidExpression
- TestCronScheduling_ExecutionTiming
- TestCronScheduling_ErrorHandling
- TestCronScheduling_ContextCancellation
- TestBackwardCompatibility (ensure existing tests pass)
```

### Manual Testing
```bash
# Test interval mode (existing)
go run demo-app/cmd/scheduler/main.go

# Test cron mode (new)
go run demo-app/cmd/scheduler-cron/main.go
```

## Code Quality Checklist

- [ ] Follow Go conventions from `.agents/go-conventions.md`
  - [ ] Use camelCase for JSON tags
  - [ ] Wrap errors with fmt.Errorf
  - [ ] Define package-level error variables
  - [ ] Use interface-based dependency injection
- [ ] Follow testing conventions from `.agents/testing.md`
  - [ ] Use `_test` package suffix
  - [ ] Add `t.Parallel()` to all tests
  - [ ] Use standard library assertions (no testify)
  - [ ] Hand-roll mocks if needed
- [ ] Pass all linters: `task lint`
- [ ] Maintain test coverage
- [ ] Update relevant documentation

## Expected Outcomes

### API Examples

**Before (interval-only):**
```go
s := scheduler.New(5*time.Minute, application.RunnerFunc(task))
```

**After (with cron support):**
```go
// Interval mode (unchanged)
s := scheduler.New(5*time.Minute, application.RunnerFunc(task))

// Cron mode (new)
s, err := scheduler.NewWithCron("*/5 * * * *", application.RunnerFunc(task))
if err != nil {
    log.Fatal(err)
}
```

### Benefits
1. **More flexible scheduling** - Users can express complex schedules (e.g., "every Monday at 9am")
2. **Industry standard** - Cron syntax is widely understood
3. **Backward compatible** - Existing code continues to work
4. **Simple API** - Easy to use with clear error handling

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing API | Keep existing constructor unchanged, add new one |
| Invalid cron expressions | Validate at construction time, return error |
| Performance overhead | Cron library is well-optimized; minimal impact |
| Increased complexity | Clear separation of concerns with mode field |

## Dependencies
- `github.com/pardnchiu/go-scheduler` - Modern cron library for Go
  - Lightweight with minimal dependencies (stdlib only)
  - Supports standard cron syntax, custom descriptors (@daily, @hourly)
  - Includes @every interval syntax (@every 5m, @every 2h)
  - Built-in task dependencies, timeouts, and panic recovery
  - MIT licensed, active development

## Acceptance Criteria
- [ ] Users can create schedulers with cron syntax
- [ ] Invalid cron expressions return clear errors at construction time
- [ ] Cron-based schedulers execute at correct times
- [ ] All existing tests pass (backward compatibility)
- [ ] New tests for cron functionality pass
- [ ] Documentation updated with cron examples
- [ ] Demo application shows cron usage
- [ ] All linters pass
- [ ] Test coverage maintained or improved
