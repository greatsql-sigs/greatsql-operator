package pkg

import (
	"fmt"
	"time"

	"slices"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
)

// StateMachine 状态机结构
type StateMachine struct {
	// currentPhase v1alpha1.Phase
	status *v1alpha1.Status
}

func NewStateMachine(status *v1alpha1.Status) *StateMachine {
	return &StateMachine{
		// currentPhase: status.Phase,
		status: status,
	}
}

// validTransitions 状态转换规则
var validTransitions = map[v1alpha1.Phase][]v1alpha1.Phase{
	v1alpha1.PhaseInitializing: {v1alpha1.PhaseRunning, v1alpha1.PhaseError, v1alpha1.PhaseReady},
	v1alpha1.PhaseRunning:      {v1alpha1.PhaseReady, v1alpha1.PhaseError, v1alpha1.PhaseStoping},
	v1alpha1.PhaseStoping:      {v1alpha1.PhaseError},
	v1alpha1.PhaseReady:        {v1alpha1.PhaseRunning, v1alpha1.PhaseError, v1alpha1.PhasePaused},
	v1alpha1.PhaseError:        {v1alpha1.PhaseInitializing},
	v1alpha1.PhasePaused:       {v1alpha1.PhaseRunning, v1alpha1.PhaseError},
}

// Transition 状态转换
func (sm *StateMachine) Transition(newPhase v1alpha1.Phase) error {
	current := sm.status.Phase

	if current == newPhase {
		return nil
	}

	allowedPhases, exists := validTransitions[current]
	if !exists {
		return fmt.Errorf("no valid transitions defined for phase: %s", current)
	}

	if !slices.Contains(allowedPhases, newPhase) {
		return fmt.Errorf("invalid state transition from %s to %s", current, newPhase)
	}

	// 执行状态切换
	sm.status.Phase = newPhase
	sm.status.Age = time.Now().Format(time.RFC3339)

	// 设置默认 Message/Reason
	switch newPhase {
	case v1alpha1.PhaseError:
		if sm.status.Message == "" {
			sm.status.Message = "system error"
		}
		if sm.status.Reason == "" {
			sm.status.Reason = "unknown error"
		}
	case v1alpha1.PhasePaused:
		if sm.status.Message == "" {
			sm.status.Message = "system paused"
		}
		if sm.status.Reason == "" {
			sm.status.Reason = "manual paused"
		}
	}

	return nil
}

// SetStatusMessage 设置状态信息
func (sm *StateMachine) SetStatusMessage(msg string) {
	sm.status.Message = msg
}

// SetStatusReason 设置状态原因
func (sm *StateMachine) SetStatusReason(reason string) {
	sm.status.Reason = reason
}

// SetReady 设置 Ready 字段
func (sm *StateMachine) SetReady(count int32) {
	sm.status.Ready = count
}

// GetStatus 获取完整 status 对象
func (sm *StateMachine) GetStatus() *v1alpha1.Status {
	return sm.status
}
