package mssql

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestNewClient проверяет создание нового клиента с различными параметрами
func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
		// Ожидаемые значения после создания клиента (с defaults)
		wantPort     int
		wantDatabase string
		wantTimeout  time.Duration
	}{
		{
			name: "пустые параметры - устанавливаются значения по умолчанию",
			opts: ClientOptions{
				Server: "test-server",
			},
			wantPort:     1433,
			wantDatabase: "master",
			wantTimeout:  30 * time.Second,
		},
		{
			name: "все параметры заданы - не меняются",
			opts: ClientOptions{
				Server:   "custom-server",
				Port:     1434,
				User:     "testuser",
				Password: "testpass",
				Database: "testdb",
				Timeout:  60 * time.Second,
			},
			wantPort:     1434,
			wantDatabase: "testdb",
			wantTimeout:  60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.opts)
			if err != nil {
				t.Fatalf("NewClient() error = %v, want nil", err)
			}

			// Приводим к конкретному типу для проверки полей
			cli, ok := c.(*client)
			if !ok {
				t.Fatal("NewClient() не вернул *client")
			}

			if cli.opts.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", cli.opts.Port, tt.wantPort)
			}
			if cli.opts.Database != tt.wantDatabase {
				t.Errorf("Database = %s, want %s", cli.opts.Database, tt.wantDatabase)
			}
			if cli.opts.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", cli.opts.Timeout, tt.wantTimeout)
			}
		})
	}
}

// TestClient_Ping проверяет метод Ping
func TestClient_Ping(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		noConnect bool // не устанавливать соединение
		wantErr   bool
	}{
		{
			name: "успешный ping",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			wantErr: false,
		},
		{
			name:      "ping без соединения",
			noConnect: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &client{
				opts: ClientOptions{Server: "test"},
			}

			if !tt.noConnect {
				db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
				if err != nil {
					t.Fatalf("ошибка создания sqlmock: %v", err)
				}
				defer db.Close()

				if tt.setupMock != nil {
					tt.setupMock(mock)
				}

				cli.db = db
			}

			err := cli.Ping(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestClient_Close проверяет метод Close
func TestClient_Close(t *testing.T) {
	t.Run("закрытие активного соединения", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("ошибка создания sqlmock: %v", err)
		}

		mock.ExpectClose()

		cli := &client{
			db:   db,
			opts: ClientOptions{Server: "test"},
		}

		if err := cli.Close(); err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}

		if cli.db != nil {
			t.Error("Close() не обнулил cli.db")
		}
	})

	t.Run("закрытие nil соединения", func(t *testing.T) {
		cli := &client{
			opts: ClientOptions{Server: "test"},
		}

		if err := cli.Close(); err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})
}

// TestClient_GetRestoreStats проверяет получение статистики восстановления
func TestClient_GetRestoreStats(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		opts      StatsOptions
		noConnect bool
		want      *RestoreStats
		wantErr   bool
	}{
		{
			name: "успешное получение статистики",
			opts: StatsOptions{
				SrcDB:           "TestDB",
				DstServer:       "TestServer",
				TimeToStatistic: "2024-01-01T00:00:00",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"avg", "max"}).
					AddRow(180, 600)
				mock.ExpectQuery("SELECT").
					WithArgs("2024-01-01T00:00:00", "TestDB", "TestServer").
					WillReturnRows(rows)
			},
			want: &RestoreStats{
				AvgRestoreTimeSec: 180,
				MaxRestoreTimeSec: 600,
				HasData:           true,
			},
			wantErr: false,
		},
		{
			name: "нет данных статистики",
			opts: StatsOptions{
				SrcDB:           "EmptyDB",
				DstServer:       "TestServer",
				TimeToStatistic: "2024-01-01T00:00:00",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"avg", "max"}).
					AddRow(nil, nil)
				mock.ExpectQuery("SELECT").
					WithArgs("2024-01-01T00:00:00", "EmptyDB", "TestServer").
					WillReturnRows(rows)
			},
			want: &RestoreStats{
				HasData: false,
			},
			wantErr: false,
		},
		{
			name: "нет строк (ErrNoRows)",
			opts: StatsOptions{
				SrcDB:           "NoDataDB",
				DstServer:       "TestServer",
				TimeToStatistic: "2024-01-01T00:00:00",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").
					WithArgs("2024-01-01T00:00:00", "NoDataDB", "TestServer").
					WillReturnError(sql.ErrNoRows)
			},
			want: &RestoreStats{
				HasData: false,
			},
			wantErr: false,
		},
		{
			name: "ошибка запроса",
			opts: StatsOptions{
				SrcDB:           "ErrorDB",
				DstServer:       "TestServer",
				TimeToStatistic: "2024-01-01T00:00:00",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").
					WithArgs("2024-01-01T00:00:00", "ErrorDB", "TestServer").
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name:      "нет соединения",
			noConnect: true,
			opts:      StatsOptions{SrcDB: "Test"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &client{
				opts: ClientOptions{Server: "test"},
			}

			if !tt.noConnect {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatalf("ошибка создания sqlmock: %v", err)
				}
				defer db.Close()

				if tt.setupMock != nil {
					tt.setupMock(mock)
				}

				cli.db = db
			}

			got, err := cli.GetRestoreStats(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRestoreStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.want != nil {
				if got.HasData != tt.want.HasData {
					t.Errorf("HasData = %v, want %v", got.HasData, tt.want.HasData)
				}
				if got.AvgRestoreTimeSec != tt.want.AvgRestoreTimeSec {
					t.Errorf("AvgRestoreTimeSec = %d, want %d", got.AvgRestoreTimeSec, tt.want.AvgRestoreTimeSec)
				}
				if got.MaxRestoreTimeSec != tt.want.MaxRestoreTimeSec {
					t.Errorf("MaxRestoreTimeSec = %d, want %d", got.MaxRestoreTimeSec, tt.want.MaxRestoreTimeSec)
				}
			}
		})
	}
}

// TestClient_Restore проверяет выполнение восстановления базы данных
func TestClient_Restore(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		opts      RestoreOptions
		noConnect bool
		wantErr   bool
	}{
		{
			name: "успешное восстановление",
			opts: RestoreOptions{
				Description:   "test restore",
				TimeToRestore: "2024-01-01T12:00:00",
				User:          "testuser",
				SrcServer:     "SrcServer",
				SrcDB:         "SrcDB",
				DstServer:     "DstServer",
				DstDB:         "DstDB",
				Timeout:       10 * time.Minute,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("USE master").
					WithArgs(
						"test restore",
						"2024-01-01T12:00:00",
						"testuser",
						"SrcServer",
						"SrcDB",
						"DstServer",
						"DstDB",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "ошибка выполнения",
			opts: RestoreOptions{
				Description: "test restore",
				SrcDB:       "SrcDB",
				DstDB:       "DstDB",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("USE master").
					WillReturnError(errors.New("restore failed"))
			},
			wantErr: true,
		},
		{
			name:      "нет соединения",
			noConnect: true,
			opts:      RestoreOptions{SrcDB: "Test"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &client{
				opts: ClientOptions{Server: "test"},
			}

			if !tt.noConnect {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatalf("ошибка создания sqlmock: %v", err)
				}
				defer db.Close()

				if tt.setupMock != nil {
					tt.setupMock(mock)
				}

				cli.db = db
			}

			err := cli.Restore(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Restore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestClient_GetBackupSize проверяет получение размера бэкапа
func TestClient_GetBackupSize(t *testing.T) {
	tests := []struct {
		name      string
		database  string
		setupMock func(mock sqlmock.Sqlmock)
		noConnect bool
		want      int64
		wantErr   bool
	}{
		{
			name:     "успешное получение размера",
			database: "TestDB",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"backup_size"}).
					AddRow(500 * 1024 * 1024) // 500 MB
				mock.ExpectQuery("SELECT TOP 1").
					WithArgs("TestDB").
					WillReturnRows(rows)
			},
			want:    500 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:     "нет бэкапов",
			database: "NewDB",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT TOP 1").
					WithArgs("NewDB").
					WillReturnError(sql.ErrNoRows)
			},
			want:    0,
			wantErr: false,
		},
		{
			name:     "NULL размер",
			database: "NullDB",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"backup_size"}).
					AddRow(nil)
				mock.ExpectQuery("SELECT TOP 1").
					WithArgs("NullDB").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: false,
		},
		{
			name:      "нет соединения",
			database:  "Test",
			noConnect: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &client{
				opts: ClientOptions{Server: "test"},
			}

			if !tt.noConnect {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatalf("ошибка создания sqlmock: %v", err)
				}
				defer db.Close()

				if tt.setupMock != nil {
					tt.setupMock(mock)
				}

				cli.db = db
			}

			got, err := cli.GetBackupSize(context.Background(), tt.database)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackupSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("GetBackupSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Compile-time проверка уже есть в client.go:15, тест удалён (L5 fix)

// TestNewClient_InvalidPort проверяет валидацию порта (M6 fix)
func TestNewClient_InvalidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{name: "порт 0 - используется default 1433", port: 0, wantErr: false},
		{name: "валидный порт 1433", port: 1433, wantErr: false},
		{name: "валидный порт 1434", port: 1434, wantErr: false},
		{name: "минимальный порт 1", port: 1, wantErr: false},
		{name: "максимальный порт 65535", port: 65535, wantErr: false},
		{name: "негативный порт", port: -1, wantErr: true},
		{name: "порт больше 65535", port: 65536, wantErr: true},
		{name: "очень большой порт", port: 100000, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(ClientOptions{
				Server: "test-server",
				Port:   tt.port,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewClientWithEncrypt проверяет создание клиента с явным указанием шифрования (H3 fix)
func TestNewClientWithEncrypt(t *testing.T) {
	tests := []struct {
		name        string
		encrypt     bool
		wantEncrypt bool
	}{
		{name: "шифрование включено", encrypt: true, wantEncrypt: true},
		{name: "шифрование отключено", encrypt: false, wantEncrypt: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClientWithEncrypt(ClientOptions{Server: "test"}, tt.encrypt)
			if err != nil {
				t.Fatalf("NewClientWithEncrypt() error = %v", err)
			}

			cli, ok := c.(*client)
			if !ok {
				t.Fatal("NewClientWithEncrypt() не вернул *client")
			}

			if cli.opts.Encrypt != tt.wantEncrypt {
				t.Errorf("Encrypt = %v, want %v", cli.opts.Encrypt, tt.wantEncrypt)
			}
		})
	}
}

// TestNewClientWithEncrypt_InvalidPort проверяет валидацию порта через NewClientWithEncrypt (M3 fix)
func TestNewClientWithEncrypt_InvalidPort(t *testing.T) {
	_, err := NewClientWithEncrypt(ClientOptions{
		Server: "test-server",
		Port:   -1, // невалидный порт
	}, true)
	if err == nil {
		t.Error("NewClientWithEncrypt() должен вернуть ошибку для невалидного порта")
	}
}

// TestNewClient_DefaultEncrypt проверяет что шифрование включено по умолчанию (H3 fix)
func TestNewClient_DefaultEncrypt(t *testing.T) {
	c, err := NewClient(ClientOptions{Server: "test"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	cli, ok := c.(*client)
	if !ok {
		t.Fatal("NewClient() не вернул *client")
	}

	// По умолчанию шифрование должно быть включено
	if !cli.opts.Encrypt {
		t.Error("По умолчанию Encrypt должен быть true для безопасности")
	}
}

// TestNewClient_EmptyServer проверяет валидацию пустого Server (M5 fix)
func TestNewClient_EmptyServer(t *testing.T) {
	_, err := NewClient(ClientOptions{
		Server: "", // пустой сервер
		Port:   1433,
	})
	if err == nil {
		t.Error("NewClient() должен вернуть ошибку для пустого Server")
	}
}

// TestEscapeConnStringParam проверяет экранирование параметров (H1 fix)
func TestEscapeConnStringParam(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "простая строка", input: "password", want: "password"},
		{name: "точка с запятой", input: "pass;word", want: "pass%3Bword"},
		{name: "знак равенства", input: "pass=word", want: "pass%3Dword"},
		{name: "пробел", input: "pass word", want: "pass+word"},
		{name: "комбинация спецсимволов", input: "p@ss;w=rd!", want: "p%40ss%3Bw%3Drd%21"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeConnStringParam(tt.input)
			if got != tt.want {
				t.Errorf("escapeConnStringParam(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
