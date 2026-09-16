package git

type MockClient struct {
	SwitchToBranchFunc             func(slug string) error
	SwitchToLoopBranchFunc         func(slug string) error
	BlockedFileExistsFunc          func() bool
	WriteBlockedFileFunc           func(err error)
	HasChangesFunc                 func() bool
	ReportExistsFunc               func() bool
	CommitFromReportFunc           func(slug string) error
	CurrentBranchFunc              func() (string, error)
	IsBranchSyncedWithRemoteFunc   func(branch string) error
	CommitGeneratedArtifactsFunc   func(slug string) error
	CommitProjectRemovalFunc       func(path string) error
	FetchBranchFunc                func(branch string) error
	NeedsMergeFunc                 func(branch string) (bool, error)
	MergeFunc                      func(branch string) error
	AbortMergeFunc                 func() error
	PushFunc                       func() error
}

func (m *MockClient) SwitchToBranch(slug string) error {
	if m.SwitchToBranchFunc != nil {
		return m.SwitchToBranchFunc(slug)
	}
	return nil
}

func (m *MockClient) SwitchToLoopBranch(slug string) error {
	if m.SwitchToLoopBranchFunc != nil {
		return m.SwitchToLoopBranchFunc(slug)
	}
	return nil
}

func (m *MockClient) BlockedFileExists() bool {
	if m.BlockedFileExistsFunc != nil {
		return m.BlockedFileExistsFunc()
	}
	return false
}

func (m *MockClient) WriteBlockedFile(err error) {
	if m.WriteBlockedFileFunc != nil {
		m.WriteBlockedFileFunc(err)
	}
}

func (m *MockClient) HasChanges() bool {
	if m.HasChangesFunc != nil {
		return m.HasChangesFunc()
	}
	return false
}

func (m *MockClient) ReportExists() bool {
	if m.ReportExistsFunc != nil {
		return m.ReportExistsFunc()
	}
	return false
}

func (m *MockClient) CommitFromReport(slug string) error {
	if m.CommitFromReportFunc != nil {
		return m.CommitFromReportFunc(slug)
	}
	return nil
}

func (m *MockClient) CurrentBranch() (string, error) {
	if m.CurrentBranchFunc != nil {
		return m.CurrentBranchFunc()
	}
	return "main", nil
}

func (m *MockClient) IsBranchSyncedWithRemote(branch string) error {
	if m.IsBranchSyncedWithRemoteFunc != nil {
		return m.IsBranchSyncedWithRemoteFunc(branch)
	}
	return nil
}

func (m *MockClient) CommitGeneratedArtifacts(slug string) error {
	if m.CommitGeneratedArtifactsFunc != nil {
		return m.CommitGeneratedArtifactsFunc(slug)
	}
	return nil
}

func (m *MockClient) CommitProjectRemoval(path string) error {
	if m.CommitProjectRemovalFunc != nil {
		return m.CommitProjectRemovalFunc(path)
	}
	return nil
}

func (m *MockClient) FetchBranch(branch string) error {
	if m.FetchBranchFunc != nil {
		return m.FetchBranchFunc(branch)
	}
	return nil
}

func (m *MockClient) NeedsMerge(branch string) (bool, error) {
	if m.NeedsMergeFunc != nil {
		return m.NeedsMergeFunc(branch)
	}
	return false, nil
}

func (m *MockClient) Merge(branch string) error {
	if m.MergeFunc != nil {
		return m.MergeFunc(branch)
	}
	return nil
}

func (m *MockClient) AbortMerge() error {
	if m.AbortMergeFunc != nil {
		return m.AbortMergeFunc()
	}
	return nil
}

func (m *MockClient) Push() error {
	if m.PushFunc != nil {
		return m.PushFunc()
	}
	return nil
}
