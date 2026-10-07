package ui

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	appplugin "wox/plugin/system/app"
	"wox/ui/contract"
	"wox/util"
	"wox/util/keyboard"
)

// HotkeyAppCandidates returns platform application identities suitable for exclusion rules.
func (s *CoreServices) HotkeyAppCandidates(ctx context.Context, sessionID string) ([]contract.HotkeyApp, error) {
	apps := appplugin.GetHotkeyAppCandidates(uiServiceContext(ctx, sessionID))
	converted := make([]contract.HotkeyApp, len(apps))
	for index, app := range apps {
		converted[index] = contract.HotkeyApp{Name: app.Name, Identity: app.Identity, Path: app.Path, Icon: app.Icon}
	}
	return converted, nil
}

// IndexedApps returns applications matching the core ignore rule, including distinct shortcuts.
func (s *CoreServices) IndexedApps(ctx context.Context, sessionID string, pattern string) ([]contract.HotkeyApp, error) {
	apps := appplugin.GetIndexedApps(uiServiceContext(ctx, sessionID), pattern)
	converted := make([]contract.HotkeyApp, len(apps))
	for index, app := range apps {
		converted[index] = contract.HotkeyApp{Name: app.Name, Identity: app.Identity, Path: app.Path, Icon: app.Icon}
	}
	return converted, nil
}

// StartHotkeyRecording activates the strongest recorder supported by the current platform.
func (s *CoreServices) StartHotkeyRecording(ctx context.Context, sessionID string, purpose string, allowedKinds []string) (contract.HotkeyRecordingCapability, error) {
	purpose = strings.TrimSpace(purpose)
	if purpose == "" {
		return contract.HotkeyRecordingCapability{}, errors.New("purpose is required when recording starts")
	}
	if len(allowedKinds) == 0 {
		return contract.HotkeyRecordingCapability{}, errors.New("allowedKinds is required when recording starts")
	}
	ctx = uiServiceContext(ctx, sessionID)
	logger.Info(ctx, fmt.Sprintf("received hotkey recording state from UI: isRecording=true purpose=%s allowedKinds=%v", purpose, allowedKinds))
	if runtime.GOOS == "darwin" {
		// Pending raw subscriptions are valid for global hotkeys, but cannot record
		// input yet. Probe again on each attempt so granting access allows a retry.
		status, err := s.MacOSPermissionStatus(ctx, sessionID)
		util.GetLogger().Info(ctx, fmt.Sprintf("hotkey recording permission probe: session=%s status=%+v error=%v", sessionID, status, err))
		if err != nil {
			return contract.HotkeyRecordingCapability{}, err
		}
		granted := status.Accessibility == "granted"
		if err := keyboard.ReconcileRawKeyListenerAccessWithPermissionStatus(granted); err != nil {
			return contract.HotkeyRecordingCapability{}, err
		}
		if !granted {
			_, _ = GetUIManager().PostOnHotkeyRecording(ctx, false, "", nil)
			util.GetLogger().Warn(ctx, "hotkey recording blocked: Accessibility permission is required")
			return contract.HotkeyRecordingCapability{}, errors.New("i18n:ui_hotkey_accessibility_required")
		}
	}
	capability, err := GetUIManager().PostOnHotkeyRecording(ctx, true, purpose, allowedKinds)
	if err != nil {
		return contract.HotkeyRecordingCapability{}, err
	}
	return contract.HotkeyRecordingCapability{
		RawRecorderAvailable: capability.RawRecorderAvailable,
		FallbackAllowedKinds: append([]string(nil), capability.FallbackAllowedKinds...),
		UnavailableReason:    capability.UnavailableReason,
	}, nil
}

// StopHotkeyRecording releases the process-wide recorder.
func (s *CoreServices) StopHotkeyRecording(ctx context.Context, sessionID string) error {
	ctx = uiServiceContext(ctx, sessionID)
	logger.Info(ctx, "received hotkey recording state from UI: isRecording=false")
	_, err := GetUIManager().PostOnHotkeyRecording(ctx, false, "", nil)
	return err
}

// SubmitHotkeyRecordingCandidate forwards a locally parsed normal combo to the raw recorder path.
func (s *CoreServices) SubmitHotkeyRecordingCandidate(ctx context.Context, sessionID string, hotkey string) error {
	hotkey = strings.TrimSpace(hotkey)
	if hotkey == "" {
		return errors.New("hotkey is required")
	}
	return GetUIManager().PostHotkeyRecordingCandidate(uiServiceContext(ctx, sessionID), hotkey)
}

// CheckHotkeyAvailability checks Wox-owned and operating-system conflicts.
func (s *CoreServices) CheckHotkeyAvailability(ctx context.Context, sessionID string, hotkey string) (contract.HotkeyAvailability, error) {
	hotkey = strings.TrimSpace(hotkey)
	if hotkey == "" {
		return contract.HotkeyAvailability{}, errors.New("hotkey is empty")
	}
	availability := GetUIManager().CheckHotkeyAvailability(uiServiceContext(ctx, sessionID), hotkey)
	return contract.HotkeyAvailability{
		Available: availability.Available, ConflictType: availability.ConflictType, ConflictValue: availability.ConflictValue,
	}, nil
}
