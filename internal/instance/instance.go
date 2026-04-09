package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	defaultBaseDirName = ".flarness"
	instancesDirName   = "instances"
)

// Paths describes the filesystem layout for a single flarness session.
type Paths struct {
	BaseDir       string
	InstancesDir  string
	InstanceDir   string
	SocketPath    string
	PIDPath       string
	DaemonLogPath string
	MetaPath      string
	LogsDir       string
}

// Meta stores persisted metadata for a daemon instance.
type Meta struct {
	Session     string `json:"session"`
	ProjectPath string `json:"project_path"`
	ProjectName string `json:"project_name"`
	Device      string `json:"device"`
	CreatedAt   string `json:"created_at"`
	Version     string `json:"version,omitempty"`
}

// BaseDir returns the flarness home directory.
func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultBaseDirName)
}

// InstancesDir returns the directory containing all daemon instances.
func InstancesDir() string {
	return filepath.Join(BaseDir(), instancesDirName)
}

// PathsForSession returns the filesystem layout for a session.
func PathsForSession(session string) Paths {
	baseDir := BaseDir()
	instancesDir := filepath.Join(baseDir, instancesDirName)
	instanceDir := filepath.Join(instancesDir, session)
	return Paths{
		BaseDir:       baseDir,
		InstancesDir:  instancesDir,
		InstanceDir:   instanceDir,
		SocketPath:    filepath.Join(instanceDir, "daemon.sock"),
		PIDPath:       filepath.Join(instanceDir, "daemon.pid"),
		DaemonLogPath: filepath.Join(instanceDir, "daemon.log"),
		MetaPath:      filepath.Join(instanceDir, "meta.json"),
		LogsDir:       filepath.Join(instanceDir, "logs"),
	}
}

// SessionForProject returns a stable session id for a project path.
func SessionForProject(project string) string {
	return fnvHash(project)
}

// LoadMeta reads metadata for a session from disk.
func LoadMeta(session string) (Meta, error) {
	return LoadMetaFromPath(PathsForSession(session).MetaPath)
}

// LoadMetaFromPath reads metadata from a specific file path.
func LoadMetaFromPath(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

// SaveMeta writes metadata for a session to disk.
func SaveMeta(meta Meta) error {
	paths := PathsForSession(meta.Session)
	if err := os.MkdirAll(paths.InstanceDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.MetaPath, data, 0644)
}

// ListMetas lists all instances with metadata present on disk.
func ListMetas() ([]Meta, error) {
	entries, err := os.ReadDir(InstancesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []Meta{}, nil
		}
		return nil, err
	}

	metas := make([]Meta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := LoadMeta(entry.Name())
		if err != nil {
			continue
		}
		if meta.Session == "" {
			meta.Session = entry.Name()
		}
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Session < metas[j].Session
	})
	return metas, nil
}

// Cleanup removes transient files and optionally the whole instance directory if empty.
func Cleanup(session string) error {
	paths := PathsForSession(session)
	_ = os.Remove(paths.SocketPath)
	_ = os.Remove(paths.PIDPath)

	if err := removeIfEmpty(paths.LogsDir); err != nil {
		return err
	}
	return removeIfEmpty(paths.InstanceDir)
}

// CleanupAll removes the full instance directory tree.
func CleanupAll(session string) error {
	return os.RemoveAll(PathsForSession(session).InstanceDir)
}

func removeIfEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) != 0 {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func fnvHash(s string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}
