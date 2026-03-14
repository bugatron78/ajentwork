package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ajentwork/internal/domain"
	"ajentwork/internal/idgen"
)

type AttachArtifactOptions struct {
	RepoPath string
	ItemID   string
	Path     string
	Summary  string
	Label    string
}

type RecordReceiptOptions struct {
	RepoPath string
	ItemID   string
	Summary  string
	Command  string
	ExitCode int
	Output   string
	Label    string
}

func AttachArtifact(opts AttachArtifactOptions) (domain.Artifact, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Artifact{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Path) == "" {
		return domain.Artifact{}, errors.New("path is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Artifact{}, errors.New("summary is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Artifact{}, err
	}

	sourcePath, err := filepath.Abs(opts.Path)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("resolve artifact path: %w", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("stat artifact path: %w", err)
	}
	if info.IsDir() {
		return domain.Artifact{}, fmt.Errorf("artifact path %s is a directory; attach a file instead", sourcePath)
	}

	artifact, err := createArtifactRecord(artifactRecordOptions{
		repoPath:     opts.RepoPath,
		item:         &item,
		itemDir:      itemDir,
		kind:         domain.ArtifactKindFile,
		summary:      strings.TrimSpace(opts.Summary),
		label:        strings.TrimSpace(opts.Label),
		originalPath: sourcePath,
		copyPath:     sourcePath,
		eventType:    "artifact_attached",
		eventSummary: fmt.Sprintf("attached artifact: %s", strings.TrimSpace(opts.Summary)),
	})
	if err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func RecordReceipt(opts RecordReceiptOptions) (domain.Artifact, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Artifact{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Artifact{}, errors.New("summary is required")
	}
	if strings.TrimSpace(opts.Command) == "" {
		return domain.Artifact{}, errors.New("command is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Artifact{}, err
	}

	outputPath := ""
	if strings.TrimSpace(opts.Output) != "" {
		outputPath, err = filepath.Abs(opts.Output)
		if err != nil {
			return domain.Artifact{}, fmt.Errorf("resolve receipt output path: %w", err)
		}
		info, err := os.Stat(outputPath)
		if err != nil {
			return domain.Artifact{}, fmt.Errorf("stat receipt output path: %w", err)
		}
		if info.IsDir() {
			return domain.Artifact{}, fmt.Errorf("receipt output path %s is a directory; attach a file instead", outputPath)
		}
	}

	exitCode := opts.ExitCode
	artifact, err := createArtifactRecord(artifactRecordOptions{
		repoPath:     opts.RepoPath,
		item:         &item,
		itemDir:      itemDir,
		kind:         domain.ArtifactKindReceipt,
		summary:      strings.TrimSpace(opts.Summary),
		label:        strings.TrimSpace(opts.Label),
		command:      strings.TrimSpace(opts.Command),
		exitCode:     &exitCode,
		originalPath: outputPath,
		copyPath:     outputPath,
		eventType:    "receipt_recorded",
		eventSummary: fmt.Sprintf("recorded receipt: %s", strings.TrimSpace(opts.Summary)),
	})
	if err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func ListArtifacts(repoPath, itemID string, limit int) ([]domain.Artifact, error) {
	ajDir, err := ensureAJRepo(repoPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(itemID) == "" {
		return nil, errors.New("item id is required")
	}

	artifactsDir := filepath.Join(ajDir, "issues", itemID, "artifacts")
	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read artifacts directory: %w", err)
	}

	artifacts := make([]domain.Artifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		path := filepath.Join(artifactsDir, entry.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read artifact metadata: %w", err)
		}
		artifact, err := parseArtifact(string(bytes))
		if err != nil {
			return nil, fmt.Errorf("parse artifact metadata: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].CreatedAt.Equal(artifacts[j].CreatedAt) {
			return artifacts[i].ID > artifacts[j].ID
		}
		return artifacts[i].CreatedAt.After(artifacts[j].CreatedAt)
	})

	if limit > 0 && len(artifacts) > limit {
		artifacts = artifacts[:limit]
	}
	return artifacts, nil
}

type artifactRecordOptions struct {
	repoPath     string
	item         *domain.Item
	itemDir      string
	kind         domain.ArtifactKind
	summary      string
	label        string
	command      string
	exitCode     *int
	originalPath string
	copyPath     string
	eventType    string
	eventSummary string
}

func createArtifactRecord(opts artifactRecordOptions) (domain.Artifact, error) {
	artifactID, err := idgen.NewArtifactID()
	if err != nil {
		return domain.Artifact{}, err
	}

	ajDir, err := ensureAJRepo(opts.repoPath)
	if err != nil {
		return domain.Artifact{}, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	if !now.After(opts.item.UpdatedAt) {
		now = opts.item.UpdatedAt.Add(time.Second)
	}
	artifact := domain.Artifact{
		ID:           artifactID,
		ItemID:       opts.item.ID,
		Kind:         opts.kind,
		Summary:      opts.summary,
		Label:        opts.label,
		OriginalPath: opts.originalPath,
		Command:      opts.command,
		ExitCode:     opts.exitCode,
		CreatedAt:    now,
		Actor:        "agent",
	}

	if strings.TrimSpace(opts.copyPath) != "" {
		storedPath, err := copyArtifactPayload(ajDir, opts.item.ID, artifactID, opts.copyPath)
		if err != nil {
			return domain.Artifact{}, err
		}
		artifact.StoredPath = storedPath
	}

	metadataDir := filepath.Join(opts.itemDir, "artifacts")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		return domain.Artifact{}, fmt.Errorf("create artifact metadata directory: %w", err)
	}
	metadataPath := filepath.Join(metadataDir, fmt.Sprintf("%s.toml", artifact.ID))
	if err := os.WriteFile(metadataPath, []byte(marshalArtifact(artifact)), 0o644); err != nil {
		return domain.Artifact{}, fmt.Errorf("write artifact metadata: %w", err)
	}

	opts.item.UpdatedAt = now
	if err := persistItemMutationWithEventSummary(opts.itemDir, *opts.item, opts.eventType, "agent", opts.eventSummary); err != nil {
		return domain.Artifact{}, err
	}

	return artifact, nil
}

func copyArtifactPayload(ajDir, itemID, artifactID, sourcePath string) (string, error) {
	basename := filepath.Base(sourcePath)
	targetDir := filepath.Join(ajDir, "artifacts", itemID)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create artifact payload directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, fmt.Sprintf("%s_%s", artifactID, basename))
	src, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("open artifact source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("create artifact payload: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("copy artifact payload: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", fmt.Errorf("close artifact payload: %w", err)
	}

	return targetPath, nil
}

func marshalArtifact(artifact domain.Artifact) string {
	lines := []string{
		fmt.Sprintf("id = %s", strconv.Quote(artifact.ID)),
		fmt.Sprintf("item_id = %s", strconv.Quote(artifact.ItemID)),
		fmt.Sprintf("kind = %s", strconv.Quote(string(artifact.Kind))),
		fmt.Sprintf("summary = %s", strconv.Quote(artifact.Summary)),
		fmt.Sprintf("label = %s", strconv.Quote(artifact.Label)),
		fmt.Sprintf("original_path = %s", strconv.Quote(artifact.OriginalPath)),
		fmt.Sprintf("stored_path = %s", strconv.Quote(artifact.StoredPath)),
		fmt.Sprintf("command = %s", strconv.Quote(artifact.Command)),
		fmt.Sprintf("created_at = %s", strconv.Quote(artifact.CreatedAt.Format(time.RFC3339))),
		fmt.Sprintf("actor = %s", strconv.Quote(artifact.Actor)),
	}
	if artifact.ExitCode != nil {
		lines = append(lines, fmt.Sprintf("exit_code = %d", *artifact.ExitCode))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func parseArtifact(raw string) (domain.Artifact, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return domain.Artifact{}, fmt.Errorf("invalid artifact metadata line %q", line)
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	requiredString := func(key string) (string, error) {
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("missing %s", key)
		}
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid quoted value for %s: %w", key, err)
		}
		return unquoted, nil
	}

	id, err := requiredString("id")
	if err != nil {
		return domain.Artifact{}, err
	}
	itemID, err := requiredString("item_id")
	if err != nil {
		return domain.Artifact{}, err
	}
	kindRaw, err := requiredString("kind")
	if err != nil {
		return domain.Artifact{}, err
	}
	summary, err := requiredString("summary")
	if err != nil {
		return domain.Artifact{}, err
	}
	label, err := requiredString("label")
	if err != nil {
		return domain.Artifact{}, err
	}
	originalPath, err := requiredString("original_path")
	if err != nil {
		return domain.Artifact{}, err
	}
	storedPath, err := requiredString("stored_path")
	if err != nil {
		return domain.Artifact{}, err
	}
	command, err := requiredString("command")
	if err != nil {
		return domain.Artifact{}, err
	}
	actor, err := requiredString("actor")
	if err != nil {
		return domain.Artifact{}, err
	}
	createdAtRaw, err := requiredString("created_at")
	if err != nil {
		return domain.Artifact{}, err
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("invalid created_at: %w", err)
	}

	var exitCode *int
	if rawExitCode, ok := values["exit_code"]; ok {
		parsed, err := strconv.Atoi(rawExitCode)
		if err != nil {
			return domain.Artifact{}, fmt.Errorf("invalid exit_code: %w", err)
		}
		exitCode = &parsed
	}

	return domain.Artifact{
		ID:           id,
		ItemID:       itemID,
		Kind:         domain.ArtifactKind(kindRaw),
		Summary:      summary,
		Label:        label,
		OriginalPath: originalPath,
		StoredPath:   storedPath,
		Command:      command,
		ExitCode:     exitCode,
		CreatedAt:    createdAt,
		Actor:        actor,
	}, nil
}
