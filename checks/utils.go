package checks

import (
	"fmt"
	"sync"
	"time"

	"github.com/flanksource/canary-checker/api/external"
	"github.com/flanksource/canary-checker/pkg"
)

func unexpectedErrorf(check external.Check, err error, msg string, args ...interface{}) *pkg.CheckResult { //nolint: unparam
	return &pkg.CheckResult{
		Check:   check,
		Pass:    false,
		Invalid: false,
		Message: fmt.Sprintf("unexpected error %s: %v", fmt.Sprintf(msg, args...), err),
	}
}

func Error(check external.Check, err error) *pkg.CheckResult {
	return &pkg.CheckResult{
		Check:   check,
		Pass:    false,
		Invalid: true,
		Error:   err.Error(),
	}
}

func Failf(check external.Check, msg string, args ...interface{}) *pkg.CheckResult {
	return &pkg.CheckResult{
		Check:   check,
		Pass:    false,
		Invalid: false,
		Message: fmt.Sprintf(msg, args...),
	}
}

// TextFailf used for failure in case of text based results
func TextFailf(check external.Check, textResults bool, msg string, args ...interface{}) *pkg.CheckResult {
	if textResults {
		return &pkg.CheckResult{
			Check:       check,
			Pass:        false,
			Invalid:     false,
			DisplayType: "Text",
			Message:     fmt.Sprintf(msg, args...),
		}
	}
	return Failf(check, msg, args...)
}
func Success(check external.Check, start time.Time) *pkg.CheckResult {
	return &pkg.CheckResult{
		Check:    check,
		Pass:     true,
		Invalid:  false,
		Duration: time.Since(start).Milliseconds(),
	}
}

func Successf(check external.Check, start time.Time, textResults bool, msg string, args ...interface{}) *pkg.CheckResult {
	if textResults {
		return &pkg.CheckResult{
			Check:       check,
			Pass:        true,
			DisplayType: "Text",
			Invalid:     false,
			Message:     fmt.Sprintf(msg, args...),
			Duration:    time.Since(start).Milliseconds(),
		}
	}
	return &pkg.CheckResult{
		Check:    check,
		Pass:     true,
		Invalid:  false,
		Message:  fmt.Sprintf(msg, args...),
		Duration: time.Since(start).Milliseconds(),
	}
}

func Passf(check external.Check, msg string, args ...interface{}) *pkg.CheckResult {
	return &pkg.CheckResult{
		Check:   check,
		Pass:    true,
		Invalid: false,
		Message: fmt.Sprintf(msg, args...),
	}
}

type NameGenerator struct {
	NamespacesCount int
	PodsCount       int
	namespaceIndex  int
	podIndex        int
	mtx             sync.Mutex
}

func (n *NameGenerator) NamespaceName(prefix string) string {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	name := fmt.Sprintf("%s%d", prefix, n.namespaceIndex)
	n.namespaceIndex = (n.namespaceIndex + 1) % n.NamespacesCount
	return name
}

func (n *NameGenerator) PodName(prefix string) string {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	name := fmt.Sprintf("%s%d", prefix, n.podIndex)
	n.podIndex = (n.PodsCount + 1) % n.PodsCount
	return name
}

func mb(bytes int64) string {
	if bytes > 1024*1024*1024 {
		return fmt.Sprintf("%dGB", bytes/1024/1024/1024)
	} else if bytes > 1024*1024 {
		return fmt.Sprintf("%dMB", bytes/1024/1024)
	} else if bytes > 1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	}
	return fmt.Sprintf("%dB", bytes)
}
