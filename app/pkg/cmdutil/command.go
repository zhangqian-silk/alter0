package cmdutil

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
)

type CommandLogContext struct {
	SessionID string
	Stage     string
}

type commandLogContextKey struct{}

func WithCommandLogContext(ctx context.Context, meta CommandLogContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, commandLogContextKey{}, meta)
}

func getCommandLogContext(ctx context.Context) CommandLogContext {
	if ctx == nil {
		return CommandLogContext{}
	}
	meta, _ := ctx.Value(commandLogContextKey{}).(CommandLogContext)
	return meta
}

func RequireExecutable(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("missing executable")
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("executable not found: %s", name)
	}
	return nil
}

func Exists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func RunWithInput(ctx context.Context, name string, args []string, input string, timeout time.Duration) (string, error) {
	printCommand(ctx, name, args)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(execCtx, name, args...)
	if strings.TrimSpace(input) != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	output, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(decodeOutput(output))
	if err != nil {
		if outStr == "" {
			return "", formatCommandError(err, "")
		}
		return "", formatCommandError(err, outStr)
	}
	return outStr, nil
}

func Run(ctx context.Context, name string, args []string, timeout time.Duration) (string, error) {
	return RunWithInput(ctx, name, args, "", timeout)
}

func RunInteractive(ctx context.Context, name string, args []string, timeout time.Duration) error {
	printCommand(ctx, name, args)
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(execCtx, name, args...)
	cmd.Stdin = os.Stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	done := make(chan struct{}, 2)
	go readStream(stdout, &outBuf, os.Stdout, done)
	go readStream(stderr, &errBuf, os.Stderr, done)
	waitErr := cmd.Wait()
	<-done
	<-done
	combined := strings.TrimSpace(strings.TrimSuffix(outBuf.String()+"\n"+errBuf.String(), "\n"))
	if waitErr != nil {
		return formatCommandError(waitErr, combined)
	}
	if combined != "" {
		printLimitedOutput(combined)
	}
	return nil
}

func decodeOutput(data []byte) string {
	if runtime.GOOS != "windows" {
		return string(data)
	}
	if looksLikeUTF16LE(data) {
		return decodeUTF16LE(data)
	}
	return string(data)
}

func looksLikeUTF16LE(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	zeroCount := 0
	for _, b := range data {
		if b == 0x00 {
			zeroCount++
		}
	}
	return zeroCount >= len(data)/4
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 == 1 {
		data = data[:len(data)-1]
	}
	if len(data) == 0 {
		return ""
	}
	u16 := make([]uint16, len(data)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16))
}

func printCommand(ctx context.Context, name string, args []string) {
	meta := getCommandLogContext(ctx)
	if strings.TrimSpace(meta.SessionID) != "" || strings.TrimSpace(meta.Stage) != "" {
		fmt.Fprintf(os.Stderr, "[exec][sid:%s][stage:%s] %s", strings.TrimSpace(meta.SessionID), strings.TrimSpace(meta.Stage), name)
	} else {
		fmt.Fprintf(os.Stderr, "[exec] %s", name)
	}
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, " %s", strings.Join(args, " "))
	}
	fmt.Fprintln(os.Stderr)
}

func formatCommandError(err error, output string) error {
	if err == nil {
		return nil
	}
	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	if strings.TrimSpace(output) != "" {
		trimmed, truncated := limitOutputLines(output, 8)
		if truncated {
			return fmt.Errorf("exit code %d: %s\n[输出已截断，仅保留最后 8 行]", exitCode, trimmed)
		}
		return fmt.Errorf("exit code %d: %s", exitCode, trimmed)
	}
	return fmt.Errorf("exit code %d: %v", exitCode, err)
}

func readStream(r io.Reader, buf *bytes.Buffer, liveWriter io.Writer, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	tmp := make([]byte, 4096)
	var pending string
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			chunk := tmp[:n]
			buf.Write(chunk)
			pending += string(chunk)
			lines := strings.Split(pending, "\n")
			for i := 0; i < len(lines)-1; i++ {
				line := lines[i]
				if isPromptText(line) {
					_, _ = io.WriteString(liveWriter, line+"\n")
				}
			}
			pending = lines[len(lines)-1]
			if isPromptText(pending) && !strings.Contains(pending, "\n") {
				_, _ = io.WriteString(liveWriter, pending)
				pending = ""
			}
		}
		if err != nil {
			if isPromptText(pending) {
				_, _ = io.WriteString(liveWriter, pending)
			}
			return
		}
	}
}

func isPromptText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "password") || strings.Contains(lower, "passphrase") || strings.Contains(lower, "sudo") || strings.Contains(lower, "enter") || strings.Contains(lower, "请输入") {
		return true
	}
	return strings.HasSuffix(trimmed, ":") || strings.HasSuffix(trimmed, "：") || strings.HasSuffix(trimmed, "?") || strings.HasSuffix(trimmed, "？")
}

func limitOutputLines(output string, maxLines int) (string, bool) {
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return "", false
	}
	lines := strings.Split(normalized, "\n")
	if len(lines) <= maxLines {
		return normalized, false
	}
	return strings.Join(lines[len(lines)-maxLines:], "\n"), true
}

func printLimitedOutput(output string) {
	trimmed, truncated := limitOutputLines(output, 8)
	if trimmed == "" {
		return
	}
	if truncated {
		fmt.Fprintf(os.Stderr, "%s\n[输出已截断，仅保留最后 8 行]\n", trimmed)
		return
	}
	fmt.Fprintln(os.Stderr, trimmed)
}
