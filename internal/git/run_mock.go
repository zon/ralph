package git

type MockRunAdapter struct {
	SwitchToBranchFunc    func(slug string) error
	BlockedFileExistsFunc func() bool
	WriteBlockedFileFunc  func(err error)
	HasChangesFunc        func() bool
	ReportExistsFunc      func() bool
	CommitFromReportFunc  func(slug string) error

	SwitchToBranchCalled   bool
	WriteBlockedFileCalled bool
	CommitFromReportCalled bool
}

func (m *MockRunAdapter) SwitchToBranch(slug string) error {
	m.SwitchToBranchCalled = true
	if m.SwitchToBranchFunc != nil {
		return m.SwitchToBranchFunc(slug)
	}
	return nil
}

func (m *MockRunAdapter) BlockedFileExists() bool {
	if m.BlockedFileExistsFunc != nil {
		return m.BlockedFileExistsFunc()
	}
	return false
}

func (m *MockRunAdapter) WriteBlockedFile(err error) {
	m.WriteBlockedFileCalled = true
	if m.WriteBlockedFileFunc != nil {
		m.WriteBlockedFileFunc(err)
	}
}

func (m *MockRunAdapter) HasChanges() bool {
	if m.HasChangesFunc != nil {
		return m.HasChangesFunc()
	}
	return false
}

func (m *MockRunAdapter) ReportExists() bool {
	if m.ReportExistsFunc != nil {
		return m.ReportExistsFunc()
	}
	return false
}

func (m *MockRunAdapter) CommitFromReport(slug string) error {
	m.CommitFromReportCalled = true
	if m.CommitFromReportFunc != nil {
		return m.CommitFromReportFunc(slug)
	}
	return nil
}
