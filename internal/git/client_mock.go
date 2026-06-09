package git

type MockClient struct {
	SwitchToBranchFunc           func(slug string) error
	BlockedFileExistsFunc        func() bool
	WriteBlockedFileFunc         func(err error)
	HasChangesFunc               func() bool
	ReportExistsFunc             func() bool
	CommitFromReportFunc         func(slug string) error
	CurrentBranchFunc            func() (string, error)
	IsBranchSyncedWithRemoteFunc func(branch string) error
}

func (m *MockClient) SwitchToBranch(slug string) error {
	if m.SwitchToBranchFunc != nil {
		return m.SwitchToBranchFunc(slug)
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
