package web

import (
	"strings"
	"testing"
)

func TestWorkspaceServiceProcessEnvClearsInheritedWebLoginPassword(t *testing.T) {
	t.Setenv("ALTER0_WEB_LOGIN_PASSWORD", "shared-gateway-password")
	t.Setenv("ALTER0_RUNTIME_MANAGER", "systemd")

	env := workspaceServiceProcessEnv(19191)

	if value := lookupEnvValue(env, "ALTER0_WEB_LOGIN_PASSWORD"); value != "" {
		t.Fatalf("ALTER0_WEB_LOGIN_PASSWORD = %q, want empty", value)
	}
	if value := lookupEnvValue(env, "PORT"); value != "19191" {
		t.Fatalf("PORT = %q, want 19191", value)
	}
	if value := lookupEnvValue(env, "ALTER0_SERVICE_PORT"); value != "19191" {
		t.Fatalf("ALTER0_SERVICE_PORT = %q, want 19191", value)
	}
	if value := lookupEnvValue(env, "ALTER0_RUNTIME_MANAGER"); value != "systemd" {
		t.Fatalf("ALTER0_RUNTIME_MANAGER = %q, want systemd", value)
	}
}

func lookupEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}
