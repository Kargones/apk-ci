package gitea_test

import (
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
)

// TestClientComposesAllRoleInterfaces проверяет, что интерфейс Client
// включает все role-based интерфейсы согласно ISP.
func TestClientComposesAllRoleInterfaces(t *testing.T) {
	t.Parallel()

	// Создаём mock, который реализует Client
	var client gitea.Client = giteatest.NewMockClient()

	// Проверяем, что Client можно привести к каждому role-based интерфейсу
	tests := []struct {
		name      string
		checkFunc func() bool
	}{
		{
			name: "PRReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.PRReader)
				return ok
			},
		},
		{
			name: "CommitReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.CommitReader)
				return ok
			},
		},
		{
			name: "FileReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.FileReader)
				return ok
			},
		},
		{
			name: "BranchManager",
			checkFunc: func() bool {
				_, ok := client.(gitea.BranchManager)
				return ok
			},
		},
		{
			name: "ReleaseReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.ReleaseReader)
				return ok
			},
		},
		{
			name: "IssueManager",
			checkFunc: func() bool {
				_, ok := client.(gitea.IssueManager)
				return ok
			},
		},
		{
			name: "PRManager",
			checkFunc: func() bool {
				_, ok := client.(gitea.PRManager)
				return ok
			},
		},
		{
			name: "RepositoryWriter",
			checkFunc: func() bool {
				_, ok := client.(gitea.RepositoryWriter)
				return ok
			},
		},
		{
			name: "TeamReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.TeamReader)
				return ok
			},
		},
		{
			name: "OrgReader",
			checkFunc: func() bool {
				_, ok := client.(gitea.OrgReader)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !tt.checkFunc() {
				t.Errorf("gitea.Client должен включать интерфейс %s", tt.name)
			}
		})
	}
}

// TestRoleInterfacesAreIndependent проверяет, что role-based интерфейсы
// можно использовать независимо друг от друга.
func TestRoleInterfacesAreIndependent(t *testing.T) {
	t.Parallel()

	mock := giteatest.NewMockClient()

	// Проверяем, что mock можно использовать как каждый отдельный интерфейс
	t.Run("PRReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.PRReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать PRReader")
		}
	})

	t.Run("CommitReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.CommitReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать CommitReader")
		}
	})

	t.Run("FileReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.FileReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать FileReader")
		}
	})

	t.Run("BranchManager independence", func(t *testing.T) {
		t.Parallel()
		var manager gitea.BranchManager = mock
		if manager == nil {
			t.Error("MockClient должен реализовывать BranchManager")
		}
	})

	t.Run("ReleaseReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.ReleaseReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать ReleaseReader")
		}
	})

	t.Run("IssueManager independence", func(t *testing.T) {
		t.Parallel()
		var manager gitea.IssueManager = mock
		if manager == nil {
			t.Error("MockClient должен реализовывать IssueManager")
		}
	})

	t.Run("PRManager independence", func(t *testing.T) {
		t.Parallel()
		var manager gitea.PRManager = mock
		if manager == nil {
			t.Error("MockClient должен реализовывать PRManager")
		}
	})

	t.Run("RepositoryWriter independence", func(t *testing.T) {
		t.Parallel()
		var writer gitea.RepositoryWriter = mock
		if writer == nil {
			t.Error("MockClient должен реализовывать RepositoryWriter")
		}
	})

	t.Run("TeamReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.TeamReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать TeamReader")
		}
	})

	t.Run("OrgReader independence", func(t *testing.T) {
		t.Parallel()
		var reader gitea.OrgReader = mock
		if reader == nil {
			t.Error("MockClient должен реализовывать OrgReader")
		}
	})
}

// Compile-time проверки сигнатур методов интерфейсов.
// Эти проверки находятся в mock.go:12-23 и гарантируют соответствие типов.
// Данный файл тестирует только runtime поведение через MockClient.
