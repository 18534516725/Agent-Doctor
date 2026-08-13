package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

type Command struct {
	Argv     []string `json:"argv"`
	Approved bool     `json:"approved"`
}

type Plan struct {
	Repository    string    `json:"repository"`
	BaseSHA       string    `json:"baseSha"`
	Client        string    `json:"client"`
	Model         string    `json:"model"`
	SanitizedTask string    `json:"sanitizedTask"`
	Commands      []Command `json:"commands"`
	MaxCalls      int       `json:"maxCalls"`
	MaxCostMicros int64     `json:"maxCostMicros"`
}

type Consent struct {
	PlanHash string `json:"planHash"`
}

func (plan Plan) Hash() string {
	copy := plan
	copy.SanitizedTask = privacy.FilterText(copy.SanitizedTask)
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (plan Plan) Preview() string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "Client: %s\nModel: %s\nBase SHA: %s\nTask: %s\n", plan.Client, plan.Model, plan.BaseSHA, privacy.FilterText(plan.SanitizedTask))
	for _, command := range plan.Commands {
		_, _ = fmt.Fprintf(&builder, "Command: %s\n", strings.Join(command.Argv, " "))
	}
	_, _ = fmt.Fprintf(&builder, "Max calls: %d\nMax cost: %d micros\nCleanup: temporary detached worktree is removed after execution\nPlan hash: %s\n", plan.MaxCalls, plan.MaxCostMicros, plan.Hash())
	return builder.String()
}
