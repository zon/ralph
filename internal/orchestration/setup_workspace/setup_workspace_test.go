package setup_workspace

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
)

var errMockFailure = errors.New("mock failure")

type mockFileInfo struct {
	os.FileInfo
	name string
}

type mockConfigLoader struct {
	loadFn func() (*config.RalphConfig, error)
}

func (m *mockConfigLoader) Load() (*config.RalphConfig, error) {
	if m.loadFn != nil {
		return m.loadFn()
	}
	return &config.RalphConfig{
		Workflow: config.WorkflowConfig{
			ConfigMaps: []config.ConfigMapMount{
				{Name: "cm1", DestFile: "/etc/config/file1", Link: true},
			},
			Secrets: []config.SecretMount{
				{Name: "sec1", DestDir: "/etc/secrets/dir1", Link: true},
			},
		},
	}, nil
}

type mockFsClient struct {
	getwdFn    func() (string, error)
	statFn     func(name string) (os.FileInfo, error)
	mkdirAllFn func(path string, perm os.FileMode) error
	lstatFn    func(name string) (os.FileInfo, error)
	symlinkFn  func(oldName, newName string) error
}

func (m *mockFsClient) Getwd() (string, error) {
	if m.getwdFn != nil {
		return m.getwdFn()
	}
	return "/workspace/repo", nil
}

func (m *mockFsClient) Stat(name string) (os.FileInfo, error) {
	if m.statFn != nil {
		return m.statFn(name)
	}
	return &mockFileInfo{}, nil
}

func (m *mockFsClient) MkdirAll(path string, perm os.FileMode) error {
	if m.mkdirAllFn != nil {
		return m.mkdirAllFn(path, perm)
	}
	return nil
}

func (m *mockFsClient) Lstat(name string) (os.FileInfo, error) {
	if m.lstatFn != nil {
		return m.lstatFn(name)
	}
	return nil, os.ErrNotExist
}

func (m *mockFsClient) Symlink(oldName, newName string) error {
	if m.symlinkFn != nil {
		return m.symlinkFn(oldName, newName)
	}
	return nil
}

type mockLogger struct {
	infoFn func(format string, args ...interface{})
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	if m.infoFn != nil {
		m.infoFn(format, args...)
	}
}

func newCmd(cl ConfigLoader, fs FsClient, log Logger) *SetupWorkspaceCmd {
	if cl == nil {
		cl = &mockConfigLoader{}
	}
	if fs == nil {
		fs = &mockFsClient{}
	}
	if log == nil {
		log = &mockLogger{}
	}
	return New("/workspace", cl, fs, log)
}

func TestRun_Success(t *testing.T) {
	var symlinked []string
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			symlinked = append(symlinked, newName)
			return nil
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
	require.Len(t, symlinked, 2)
	require.Equal(t, "/workspace/repo/etc/config/file1", symlinked[0])
	require.Equal(t, "/workspace/repo/etc/secrets/dir1", symlinked[1])
}

func TestRun_ConfigLoadFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(cl, nil, nil)
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_GetwdFailure(t *testing.T) {
	fs := &mockFsClient{
		getwdFn: func() (string, error) {
			return "", errMockFailure
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_LinkSrcStatFailure(t *testing.T) {
	fs := &mockFsClient{
		statFn: func(name string) (os.FileInfo, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_MkdirAllFailure(t *testing.T) {
	fs := &mockFsClient{
		mkdirAllFn: func(path string, perm os.FileMode) error {
			return errMockFailure
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_SymlinkFailure(t *testing.T) {
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_SkipExistingLink(t *testing.T) {
	fs := &mockFsClient{
		lstatFn: func(name string) (os.FileInfo, error) {
			return &mockFileInfo{}, nil
		},
		symlinkFn: func(oldName, newName string) error {
			t.Fatal("unexpected Symlink call")
			return nil
		},
	}
	cmd := newCmd(nil, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
}

func TestRun_SkipEmptyDest(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				Workflow: config.WorkflowConfig{
					ConfigMaps: []config.ConfigMapMount{
						{Name: "cm-empty", Link: true},
					},
				},
			}, nil
		},
	}
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			t.Fatal("unexpected Symlink call")
			return nil
		},
	}
	cmd := newCmd(cl, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
}

func TestRun_SkipWhenLinkFalse(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				Workflow: config.WorkflowConfig{
					ConfigMaps: []config.ConfigMapMount{
						{Name: "cm1", DestFile: "/etc/config/file1", Link: false},
					},
					Secrets: []config.SecretMount{
						{Name: "sec1", DestDir: "/etc/secrets/dir1", Link: false},
					},
				},
			}, nil
		},
	}
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			t.Fatal("unexpected Symlink call")
			return nil
		},
	}
	cmd := newCmd(cl, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
}

func TestRun_LoggedInfo(t *testing.T) {
	var logged string
	log := &mockLogger{
		infoFn: func(format string, args ...interface{}) {
			logged = format
		},
	}
	cmd := newCmd(nil, nil, log)
	err := cmd.Run()
	require.NoError(t, err)
	require.Contains(t, logged, "Linking")
}

func TestRun_DestDirFallback(t *testing.T) {
	var symlinked []string
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				Workflow: config.WorkflowConfig{
					ConfigMaps: []config.ConfigMapMount{
						{Name: "cm-dir", DestDir: "/etc/some/dir", Link: true},
					},
				},
			}, nil
		},
	}
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			symlinked = append(symlinked, newName)
			return nil
		},
	}
	cmd := newCmd(cl, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
	require.Len(t, symlinked, 1)
	require.Equal(t, "/workspace/repo/etc/some/dir", symlinked[0])
}

func TestRun_NoConfigMapsOrSecrets(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				Workflow: config.WorkflowConfig{},
			}, nil
		},
	}
	fs := &mockFsClient{
		symlinkFn: func(oldName, newName string) error {
			t.Fatal("unexpected Symlink call")
			return nil
		},
	}
	cmd := newCmd(cl, fs, nil)
	err := cmd.Run()
	require.NoError(t, err)
}
