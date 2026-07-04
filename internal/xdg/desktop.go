package xdg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const applicationsDir = "applications"

var errMissingDesktopEntry = errors.New("missing [Desktop Entry] section")

type Entry struct {
	ID       string
	Path     string
	Priority int
	Keys     map[string]string
}

type Candidate struct {
	Entry    Entry
	Selected bool
	Problems []string
}

type Result struct {
	Query      string
	ID         string
	Candidates []Candidate
}

type Options struct {
	Desktop string
}

func Lookup(query string, opts Options) (Result, error) {
	id, err := normalizeQuery(query)
	if err != nil {
		return Result{}, err
	}

	searchDirs := dataApplicationsDirs()
	var candidates []Candidate
	seen := make(map[string]bool)

	for priority, dir := range searchDirs {
		if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".desktop" {
				return nil
			}

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return nil
			}
			entryID := strings.ReplaceAll(filepath.ToSlash(rel), "/", "-")
			if entryID != id {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true

			entry, problems := readEntry(path)
			entry.ID = entryID
			entry.Path = path
			entry.Priority = priority
			problems = append(problems, visibilityProblems(entry, opts)...)
			candidates = append(candidates, Candidate{
				Entry:    entry,
				Problems: problems,
			})
			return nil
		}); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Entry.Priority < candidates[j].Entry.Priority
	})
	if len(candidates) > 0 {
		candidates[0].Selected = true
		for i := 1; i < len(candidates); i++ {
			candidates[i].Problems = append([]string{"shadowed by higher-priority desktop file"}, candidates[i].Problems...)
		}
	}

	return Result{
		Query:      query,
		ID:         id,
		Candidates: candidates,
	}, nil
}

func normalizeQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("desktop file ID or path is required")
	}

	if strings.Contains(query, string(filepath.Separator)) || strings.HasSuffix(query, ".desktop") {
		if strings.Contains(query, string(filepath.Separator)) {
			base := filepath.Base(query)
			if filepath.Ext(base) != ".desktop" {
				return "", fmt.Errorf("%q is not a .desktop file", query)
			}
			return base, nil
		}
		return query, nil
	}

	return query + ".desktop", nil
}

func dataApplicationsDirs() []string {
	homeData := envOrDefault("XDG_DATA_HOME", filepath.Join(homeDir(), ".local", "share"))
	dirs := []string{filepath.Join(homeData, applicationsDir)}

	for _, base := range splitEnvList(envOrDefault("XDG_DATA_DIRS", "/usr/local/share:/usr/share")) {
		dirs = append(dirs, filepath.Join(base, applicationsDir))
	}

	return compactDirs(dirs)
}

func readEntry(path string) (Entry, []string) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, []string{err.Error()}
	}
	defer file.Close()

	keys, err := parseDesktopEntry(file)
	if err != nil {
		return Entry{Keys: keys}, []string{err.Error()}
	}
	return Entry{Keys: keys}, nil
}

func parseDesktopEntry(r io.Reader) (map[string]string, error) {
	keys := make(map[string]string)
	inDesktopEntry := false
	foundDesktopEntry := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			inDesktopEntry = section == "Desktop Entry"
			foundDesktopEntry = foundDesktopEntry || inDesktopEntry
			continue
		}

		if !inDesktopEntry {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		keys[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	if err := scanner.Err(); err != nil {
		return keys, err
	}
	if !foundDesktopEntry {
		return keys, errMissingDesktopEntry
	}
	return keys, nil
}

func visibilityProblems(entry Entry, opts Options) []string {
	var problems []string

	if entry.Keys["Hidden"] == "true" {
		problems = append(problems, "Hidden=true")
	}
	if entry.Keys["NoDisplay"] == "true" {
		problems = append(problems, "NoDisplay=true")
	}
	if entry.Keys["Type"] != "" && entry.Keys["Type"] != "Application" {
		problems = append(problems, "Type is not Application")
	}

	desktop := strings.TrimSpace(opts.Desktop)
	if desktop == "" {
		desktop = currentDesktop()
	}
	if desktop != "" {
		desktops := splitDesktopNames(desktop)
		if only := splitDesktopNames(entry.Keys["OnlyShowIn"]); len(only) > 0 && !intersects(desktops, only) {
			problems = append(problems, "OnlyShowIn does not include current desktop")
		}
		if not := splitDesktopNames(entry.Keys["NotShowIn"]); len(not) > 0 && intersects(desktops, not) {
			problems = append(problems, "NotShowIn includes current desktop")
		}
	}

	if tryExec := strings.TrimSpace(entry.Keys["TryExec"]); tryExec != "" && !executableExists(tryExec) {
		problems = append(problems, "TryExec not found or not executable")
	}

	return problems
}

func currentDesktop() string {
	if value := os.Getenv("XDG_CURRENT_DESKTOP"); value != "" {
		return value
	}
	return os.Getenv("DESKTOP_SESSION")
}

func splitDesktopNames(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == ':' || r == ','
	})
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, strings.ToLower(field))
		}
	}
	return out
}

func intersects(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, item := range a {
		set[item] = true
	}
	for _, item := range b {
		if set[item] {
			return true
		}
	}
	return false
}

func executableExists(name string) bool {
	if strings.ContainsRune(name, filepath.Separator) {
		info, err := os.Stat(name)
		return err == nil && !info.IsDir() && hasExecBit(info.Mode())
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() && hasExecBit(info.Mode()) {
			return true
		}
	}
	return false
}

func hasExecBit(mode os.FileMode) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	return mode&0111 != 0
}

func splitEnvList(value string) []string {
	if value == "" {
		return nil
	}
	var out []string
	for _, item := range filepath.SplitList(value) {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func compactDirs(dirs []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, dir := range dirs {
		clean := filepath.Clean(dir)
		if clean != "." && !seen[clean] {
			seen[clean] = true
			out = append(out, clean)
		}
	}
	return out
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
