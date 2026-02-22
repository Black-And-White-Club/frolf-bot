package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type eventContractCatalog struct {
	Events []eventContract `json:"events"`
}

type eventContract struct {
	Subject  string        `json:"subject"`
	Producer contractActor `json:"producer"`
}

type contractActor struct {
	Service string `json:"service"`
	Module  string `json:"module"`
}

func TestDiscordEdgeContractsIncludeCoreSubjects(t *testing.T) {
	catalog := loadContractCatalog(t)
	index := make(map[string]eventContract, len(catalog.Events))
	for _, event := range catalog.Events {
		index[event.Subject] = event
	}

	expected := map[string]string{
		"discord.round.created.v1":           "discord",
		"discord.round.started.v1":           "discord",
		"round.created.v1":                   "frolf-bot-backend",
		"round.started.v1":                   "frolf-bot-backend",
		"round.participant.joined.v1":        "frolf-bot-backend",
		"round.participant.score.updated.v1": "frolf-bot-backend",
		"leaderboard.updated.v1":             "frolf-bot-backend",
		"leaderboard.tag.updated.v1":         "frolf-bot-backend",
		"leaderboard.tag.swap.processed.v1":  "frolf-bot-backend",
		"leaderboard.tag.list.requested.v1":  "pwa",
	}

	for subject, producerService := range expected {
		contract, ok := index[subject]
		if !ok {
			t.Fatalf("missing contract for subject %q", subject)
		}
		if contract.Producer.Service != producerService {
			t.Fatalf(
				"subject %q producer mismatch: expected %q got %q",
				subject,
				producerService,
				contract.Producer.Service,
			)
		}
	}
}

func loadContractCatalog(t *testing.T) eventContractCatalog {
	t.Helper()

	contractPath := os.Getenv("EVENT_CONTRACTS_PATH")
	if contractPath == "" {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatalf("failed to determine current file path")
		}
		contractPath = filepath.Clean(
			filepath.Join(filepath.Dir(filename), "../../../frolf-bot-shared/artifacts/contracts/events.v1.json"),
		)
	}

	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("failed to read contract file %q: %v", contractPath, err)
	}

	var catalog eventContractCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatalf("failed to decode contracts catalog %q: %v", contractPath, err)
	}

	return catalog
}
