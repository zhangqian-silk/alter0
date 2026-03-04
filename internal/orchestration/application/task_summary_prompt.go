package application

import (
	"fmt"
	"strings"

	taskdomain "alter0/internal/task/domain"
	tasksummaryapp "alter0/internal/tasksummary/application"
)

func buildTaskSummaryPrompt(prompt string, snapshot tasksummaryapp.Snapshot) string {
	trimmedPrompt := strings.TrimSpace(prompt)
	if len(snapshot.Summaries) == 0 {
		return trimmedPrompt
	}

	var builder strings.Builder
	builder.WriteString("[TASK SUMMARY MEMORY]\n")
	builder.WriteString("Scope: cross-session task summaries.\n")
	builder.WriteString("Retrieval mode: ")
	builder.WriteString(string(snapshot.Mode))
	builder.WriteByte('\n')
	builder.WriteString("Summaries:\n")
	for _, summary := range snapshot.Summaries {
		builder.WriteString("- task_id: ")
		builder.WriteString(strings.TrimSpace(summary.TaskID))
		builder.WriteByte('\n')
		builder.WriteString("  task_type: ")
		builder.WriteString(strings.TrimSpace(summary.TaskType))
		builder.WriteByte('\n')
		builder.WriteString("  goal: ")
		builder.WriteString(strings.TrimSpace(summary.Goal))
		builder.WriteByte('\n')
		builder.WriteString("  result: ")
		builder.WriteString(strings.TrimSpace(summary.Result))
		builder.WriteByte('\n')
		builder.WriteString("  status: ")
		builder.WriteString(string(summary.Status))
		builder.WriteByte('\n')
		builder.WriteString("  finished_at: ")
		builder.WriteString(summary.FinishedAt.UTC().Format("2006-01-02T15:04:05Z"))
		builder.WriteByte('\n')
		if len(summary.Tags) > 0 {
			builder.WriteString("  tags: ")
			builder.WriteString(strings.Join(summary.Tags, ","))
			builder.WriteByte('\n')
		}
	}

	if len(snapshot.Details) > 0 {
		builder.WriteString("Drill-down details:\n")
		for _, detail := range snapshot.Details {
			builder.WriteString("- task_id: ")
			builder.WriteString(detail.TaskID)
			builder.WriteByte('\n')
			builder.WriteString("  status: ")
			builder.WriteString(string(detail.Status))
			builder.WriteByte('\n')
			builder.WriteString("  logs:\n")
			if len(detail.Logs) == 0 {
				builder.WriteString("    - none\n")
			} else {
				for _, logItem := range detail.Logs {
					builder.WriteString("    - ")
					builder.WriteString(renderTaskLog(logItem))
					builder.WriteByte('\n')
				}
			}
			builder.WriteString("  artifacts:\n")
			if len(detail.Artifacts) == 0 {
				builder.WriteString("    - none\n")
			} else {
				for _, artifact := range detail.Artifacts {
					builder.WriteString("    - ")
					builder.WriteString(renderTaskArtifact(artifact))
					builder.WriteByte('\n')
				}
			}
		}
	} else if snapshot.Mode == tasksummaryapp.RetrievalModeDeep {
		builder.WriteString("Drill-down available by task_id: ")
		builder.WriteString(strings.Join(summaryTaskIDs(snapshot.Summaries), ","))
		builder.WriteByte('\n')
	}

	builder.WriteString("Current user input:\n")
	builder.WriteString(trimmedPrompt)
	return builder.String()
}

func summaryTaskIDs(items []taskdomain.TaskSummary) []string {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if taskID := strings.TrimSpace(item.TaskID); taskID != "" {
			ids = append(ids, taskID)
		}
	}
	return ids
}

func renderTaskLog(logItem taskdomain.TaskLog) string {
	timestamp := logItem.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
	return fmt.Sprintf("[%s][%s] %s", timestamp, logItem.Level, strings.TrimSpace(logItem.Message))
}

func renderTaskArtifact(artifact taskdomain.TaskArtifact) string {
	content := strings.TrimSpace(artifact.Content)
	if len([]rune(content)) > 120 {
		content = string([]rune(content)[:120]) + "..."
	}
	if content == "" {
		content = "no content"
	}
	return fmt.Sprintf("%s (%s): %s", strings.TrimSpace(artifact.Name), strings.TrimSpace(artifact.ContentType), content)
}
