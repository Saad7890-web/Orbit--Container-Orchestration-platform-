package docker

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Saad7890-web/orbit/internal/models"
)

var invalidNameChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func NormalizeContainerName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = invalidNameChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-_.")
	if name == "" {
		return "orbit-workload"
	}
	return name
}

func BuildContainerName(stackName, workloadName string) string {
	return fmt.Sprintf("%s-%s", NormalizeContainerName(stackName), NormalizeContainerName(workloadName))
}

func MergeLabels(base map[string]string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra)+4)
	for k, v := range extra {
		out[k] = v
	}
	for k, v := range base {
		out[k] = v
	}
	return out
}

func BaseLabels(stackName, kind, workloadName, configHash string) map[string]string {
	return map[string]string{
		LabelManaged:  "true",
		LabelStack:    stackName,
		LabelKind:     kind,
		LabelWorkload: workloadName,
		LabelHash:     configHash,
	}
}

func SortEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}

func WorkloadLabels(stackName string, kind models.WorkloadKind, workloadName, configHash string, userLabels map[string]string) map[string]string {
	base := BaseLabels(stackName, string(kind), workloadName, configHash)
	return MergeLabels(base, userLabels)
}