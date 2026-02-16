package convert

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

func getSlog(programLevel *slog.LevelVar) *slog.Logger {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	return slog.New(h)
}

func TestLoadFromConfig(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)

	l := getSlog(programLevel)

	// Создаем тестовую конфигурацию
	cfg := &config.Config{
		Owner:        "testowner",
		Repo:         "testrepo",
		ProjectName:  "TestProject",
		InfobaseName: "TEST_DB",
		AddArray:     []string{"Extension1", "Extension2"},
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Db:         "testuser",
				StoreAdmin: "admin",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Db:                 "testpass",
				StoreAdminPassword: "password123",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"TEST_DB": {
				OneServer: "testserver",
			},
		},
	}

	// Тестируем функцию LoadFromConfig
	convertConfig, err := LoadFromConfig(&ctx, l, cfg)
	if err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Проверяем StoreRoot
	expectedStoreRoot := constants.StoreRoot + "testowner/testrepo"
	if convertConfig.StoreRoot != expectedStoreRoot {
		t.Errorf("Expected StoreRoot %s, got %s", expectedStoreRoot, convertConfig.StoreRoot)
	}

	// Проверяем OneDB
	expectedDbConnectString := "/S testserver\\TEST_DB"
	if convertConfig.OneDB.DbConnectString != expectedDbConnectString {
		t.Errorf("DbConnectString = %v, want %v", convertConfig.OneDB.DbConnectString, expectedDbConnectString)
	}

	if convertConfig.OneDB.User != "testuser" {
		t.Errorf("OneDB.User = %v, want %v", convertConfig.OneDB.User, "testuser")
	}

	if convertConfig.OneDB.Pass != "testpass" {
		t.Errorf("OneDB.Pass = %v, want %v", convertConfig.OneDB.Pass, "testpass")
	}

	if !convertConfig.OneDB.ServerDb {
		t.Errorf("OneDB.ServerDb = %v, want %v", convertConfig.OneDB.ServerDb, true)
	}

	if !convertConfig.OneDB.DbExist {
		t.Errorf("OneDB.DbExist = %v, want %v", convertConfig.OneDB.DbExist, true)
	}

	// Проверяем ConvertPair
	expectedPairsCount := 3 // 1 основная + 2 расширения
	if len(convertConfig.Pair) != expectedPairsCount {
		t.Errorf("Pair count = %v, want %v", len(convertConfig.Pair), expectedPairsCount)
	}

	// Проверяем основную конфигурацию
	mainPair := convertConfig.Pair[0]
	if mainPair.Source.Name != "TestProject" {
		t.Errorf("Main Source.Name = %v, want %v", mainPair.Source.Name, "TestProject")
	}
	if mainPair.Source.RelPath != "src/cfg" {
		t.Errorf("Main Source.RelPath = %v, want %v", mainPair.Source.RelPath, "src/cfg")
	}
	if !mainPair.Source.Main {
		t.Errorf("Main Source.Main = %v, want %v", mainPair.Source.Main, true)
	}
	if mainPair.Store.Name != "TestProject" {
		t.Errorf("Main Store.Name = %v, want %v", mainPair.Store.Name, "TestProject")
	}
	if mainPair.Store.Path != "Main" {
		t.Errorf("Main Store.Path = %v, want %v", mainPair.Store.Path, "Main")
	}

	// Проверяем расширения
	for i, expectedExtName := range []string{"Extension1", "Extension2"} {
		extPair := convertConfig.Pair[i+1]
		if extPair.Source.Name != expectedExtName {
			t.Errorf("Extension[%d] Source.Name = %v, want %v", i, extPair.Source.Name, expectedExtName)
		}
		if extPair.Source.RelPath != "src/cfe/"+expectedExtName {
			t.Errorf("Extension[%d] Source.RelPath = %v, want %v", i, extPair.Source.RelPath, "src/cfe/"+expectedExtName)
		}
		if extPair.Source.Main {
			t.Errorf("Extension[%d] Source.Main = %v, want %v", i, extPair.Source.Main, false)
		}
		if extPair.Store.Path != "add/"+expectedExtName {
			t.Errorf("Extension[%d] Store.Path = %v, want %v", i, extPair.Store.Path, "add/"+expectedExtName)
		}
	}
}

func TestLoadFromConfig_DatabaseNotFound(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)

	l := getSlog(programLevel)

	// Создаем тестовую конфигурацию без нужной базы данных
	cfg := &config.Config{
		Owner:        "testowner",
		Repo:         "testrepo",
		ProjectName:  "TestProject",
		InfobaseName: "NONEXISTENT_DB",
		DbConfig:     map[string]*config.DatabaseInfo{},
	}

	// Тестируем функцию LoadFromConfig
	_, err := LoadFromConfig(&ctx, l, cfg)
	if err == nil {
		t.Fatalf("LoadFromConfig() expected error for nonexistent database, but got nil")
	}

	expectedError := "база данных NONEXISTENT_DB не найдена в конфигурации"
	if err.Error() != expectedError {
		t.Errorf("LoadFromConfig() error = %v, want %v", err.Error(), expectedError)
	}
}

/*
	func Test_Load(t *testing.T) {
		ctx := context.Background()
		var programLevel = new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)

		l := getSlog(programLevel)

		type args struct {
			ctx        *context.Context
			l          *slog.Logger
			cfg        *config.Config
			configPath string
		}
		odb := designer.OneDb{}
		cp := Pair{}
		cc := &Config{"", "", []Pair{cp, cp}, odb}
		cfg := config.MustLoad()
		cfg.TmpDir = "c:\\tmp\\4del"
		arg := args{&ctx, l, cfg}
		tests := []struct {
			name    string
			cc      *ConvertConfig
			args    args
			wantErr bool
		}{
			{
				name:    "Test1",
				cc:      cc,
				args:    arg,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.cc.Load(tt.args.ctx, tt.args.l, tt.args.cfg, "test_db"); (err != nil) != tt.wantErr {
					t.Errorf("Config.Load() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	}
*/
// func Test_Save(t *testing.T) {
// 	ctx := context.Background()
// 	var programLevel = new(slog.LevelVar)
// 	programLevel.Set(slog.LevelDebug)

// 	l := getSlog(programLevel)

// 	type args struct {
// 		ctx        *context.Context
// 		l          *slog.Logger
// 		cfg        *config.Config
// 		configPath string
// 	}
// 	odb := designer.OneDb{}
// 	cp := Pair{}
// 	cc := &Config{"", "", []Pair{cp, cp}, odb}
// 	cfg := config.MustLoad()
// 	cfg.TmpDir = "c:\\tmp\\4del"
// 	arg := args{&ctx, l, cfg, "c:\\tmp\\cc.json"}
// 	tests := []struct {
// 		name    string
// 		cc      *ConvertConfig
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name:    "Test1",
// 			cc:      cc,
// 			args:    arg,
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := tt.cc.Save(tt.args.ctx, tt.args.l, tt.args.cfg, tt.args.configPath); (err != nil) != tt.wantErr {
// 				t.Errorf("Config.Save() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

func TestLoadConfigFromData(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	// Тестовые данные JSON
	configData := []byte(`{
		"Корень хранилища": "/test/store",
		"Параметры подключения": {
			"DbConnectString": "/S testserver\\testdb",
			"User": "testuser",
			"Pass": "testpass",
			"FullConnectString": "/S testserver\\testdb /N testuser /P testpass",
			"ServerDb": true,
			"DbExist": true
		},
		"Сопоставления": [
			{
				"Источник": {
					"Имя": "TestConfig",
					"Относительный путь": "src/cfg",
					"Основная конфигурация": true
				},
				"Хранилище": {
					"Name": "TestConfig",
					"Path": "Main",
					"User": "admin",
					"Pass": "password"
				}
			}
		]
	}`)

	cfg := &config.Config{}
	convertConfig, err := LoadConfigFromData(&ctx, l, cfg, configData)
	if err != nil {
		t.Fatalf("LoadConfigFromData() error = %v", err)
	}

	// Проверяем загруженные данные
	if convertConfig.StoreRoot != "/test/store" {
		t.Errorf("StoreRoot = %v, want %v", convertConfig.StoreRoot, "/test/store")
	}

	// Функция setupDbParams может изменить пользователя, поэтому проверяем что он не пустой
	if convertConfig.OneDB.User == "" {
		t.Errorf("OneDB.User should not be empty")
	}

	if len(convertConfig.Pair) != 1 {
		t.Errorf("Pair count = %v, want %v", len(convertConfig.Pair), 1)
	}
}

func TestLoadConfigFromData_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	// Невалидные JSON данные
	configData := []byte(`{invalid json}`)

	cfg := &config.Config{}
	_, err := LoadConfigFromData(&ctx, l, cfg, configData)
	if err == nil {
		t.Fatalf("LoadConfigFromData() expected error for invalid JSON, but got nil")
	}
}

func TestConfig_Save(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	// Создаем временный файл для тестирования
	tmpFile := "/tmp/test_convert_config.json"
	defer os.Remove(tmpFile)

	// Создаем тестовую конфигурацию
	convertConfig := &Config{
		StoreRoot: "/test/store",
		OneDB: designer.OneDb{
			User: "testuser",
			Pass: "testpass",
		},
		Pair: []Pair{
			{
				Source: Source{
					Name:    "TestConfig",
					RelPath: "src/cfg",
					Main:    true,
				},
				Store: store.Store{
					Name: "TestConfig",
					Path: "Main",
				},
			},
		},
	}

	cfg := &config.Config{}
	err := convertConfig.Save(&ctx, l, cfg, tmpFile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Проверяем, что файл создан
	if !exists(tmpFile) {
		t.Errorf("Save() did not create file %s", tmpFile)
	}
}

func TestExists(t *testing.T) {
	// Тестируем существующий файл
	tmpFile := "/tmp/test_exists.txt"
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	defer os.Remove(tmpFile)

	if !exists(tmpFile) {
		t.Errorf("exists() = false, want true for existing file")
	}

	// Тестируем несуществующий файл
	if exists("/nonexistent/file.txt") {
		t.Errorf("exists() = true, want false for nonexistent file")
	}
}

func TestGetDbUser(t *testing.T) {
	// Тест с обычным пользователем
	cfg := &struct {
		DbUser     string
		DbPassword string
	}{
		DbUser: "testuser",
	}

	user := getDbUser(cfg)
	if user != "testuser" {
		t.Errorf("getDbUser() = %v, want %v", user, "testuser")
	}

	// Тест с пустым пользователем (должен вернуть значение по умолчанию)
	cfgEmpty := &struct {
		DbUser     string
		DbPassword string
	}{
		DbUser: "",
	}

	userEmpty := getDbUser(cfgEmpty)
	if userEmpty != constants.DefaultUser {
		t.Errorf("getDbUser() with empty user = %v, want %v", userEmpty, constants.DefaultUser)
	}

	// Тест с "-" (должен вернуть пустую строку)
	cfgDash := &struct {
		DbUser     string
		DbPassword string
	}{
		DbUser: "-",
	}

	userDash := getDbUser(cfgDash)
	if userDash != "" {
		t.Errorf("getDbUser() with dash = %v, want empty string", userDash)
	}
}

func TestGetDbPassword(t *testing.T) {
	// Тест с обычным паролем
	cfg := &struct {
		DbUser     string
		DbPassword string
	}{
		DbPassword: "testpass",
	}

	password := getDbPassword(cfg)
	if password != "testpass" {
		t.Errorf("getDbPassword() = %v, want %v", password, "testpass")
	}

	// Тест с пустым паролем (должен вернуть значение по умолчанию)
	cfgEmpty := &struct {
		DbUser     string
		DbPassword string
	}{
		DbPassword: "",
	}

	passwordEmpty := getDbPassword(cfgEmpty)
	if passwordEmpty != constants.DefaultPass {
		t.Errorf("getDbPassword() with empty password = %v, want %v", passwordEmpty, constants.DefaultPass)
	}

	// Тест с "-" (должен вернуть пустую строку)
	cfgDash := &struct {
		DbUser     string
		DbPassword string
	}{
		DbPassword: "-",
	}

	passwordDash := getDbPassword(cfgDash)
	if passwordDash != "" {
		t.Errorf("getDbPassword() with dash = %v, want empty string", passwordDash)
	}
}

func TestSetupDbParams(t *testing.T) {
	convertConfig := &Config{
		OneDB: designer.OneDb{
			DbConnectString: "/S testserver\\testdb",
			User:            "testuser",
			Pass:            "testpass",
		},
	}

	setupDbParams(convertConfig)

	expectedFullConnectString := "/S testserver\\testdb /N testuser /P testpass"
	if convertConfig.OneDB.FullConnectString != expectedFullConnectString {
		t.Errorf("FullConnectString = %v, want %v", convertConfig.OneDB.FullConnectString, expectedFullConnectString)
	}

	if !convertConfig.OneDB.ServerDb {
		t.Errorf("ServerDb = %v, want %v", convertConfig.OneDB.ServerDb, true)
	}

	if !convertConfig.OneDB.DbExist {
		t.Errorf("DbExist = %v, want %v", convertConfig.OneDB.DbExist, true)
	}
}

func TestMergeSetting(t *testing.T) {
	cfg := &config.Config{
		WorkDir: "/tmp",
	}

	result, err := mergeSetting(cfg)
	if err != nil {
		t.Errorf("mergeSetting() error = %v", err)
		return
	}

	// Проверяем что файл создан в правильной директории и имеет расширение .xml
	if !strings.HasPrefix(result, cfg.WorkDir) {
		t.Errorf("mergeSetting() result %v should start with %v", result, cfg.WorkDir)
	}
	if !strings.HasSuffix(result, ".xml") {
		t.Errorf("mergeSetting() result %v should end with .xml", result)
	}

	// Проверяем что файл существует
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("mergeSetting() created file %v does not exist", result)
	}

	// Очищаем созданный файл
	os.Remove(result)
}

// TestConfig_Load tests the Load method
func TestConfig_Load(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	// Test with valid project config
	cfg := &config.Config{
		Owner:       "testowner",
		Repo:        "testrepo",
		ProjectName: "TestProject",
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "TEST_DB",
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"TEST_DB": {
				OneServer: "testserver",
			},
		},
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				StoreAdmin: "admin",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				StoreAdminPassword: "password",
			},
		},
	}

	convertConfig := &Config{}
	err := convertConfig.Load(&ctx, l, cfg, "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expectedStoreRoot := constants.StoreRoot + "testowner/testrepo"
	if convertConfig.StoreRoot != expectedStoreRoot {
		t.Errorf("StoreRoot = %v, want %v", convertConfig.StoreRoot, expectedStoreRoot)
	}

	if !convertConfig.OneDB.ServerDb {
		t.Errorf("ServerDb = %v, want %v", convertConfig.OneDB.ServerDb, true)
	}
}

// TestConfig_Load_LocalBase tests the Load method with local base
func TestConfig_Load_LocalBase(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	cfg := &config.Config{
		Owner:       "testowner",
		Repo:        "testrepo",
		ProjectName: "TestProject",
		ProjectConfig: &config.ProjectConfig{
			StoreDb: constants.LocalBase,
		},
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				StoreAdmin: "admin",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				StoreAdminPassword: "password",
			},
		},
	}

	convertConfig := &Config{}
	err := convertConfig.Load(&ctx, l, cfg, "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expectedStoreRoot := constants.StoreRoot + "testowner/testrepo"
	if convertConfig.StoreRoot != expectedStoreRoot {
		t.Errorf("StoreRoot = %v, want %v", convertConfig.StoreRoot, expectedStoreRoot)
	}
}

// TestConfig_Load_DatabaseNotFound tests the Load method with nonexistent database
func TestConfig_Load_DatabaseNotFound(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	cfg := &config.Config{
		Owner:       "testowner",
		Repo:        "testrepo",
		ProjectName: "TestProject",
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "NONEXISTENT_DB",
		},
		DbConfig: map[string]*config.DatabaseInfo{},
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				StoreAdmin: "admin",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				StoreAdminPassword: "password",
			},
		},
	}

	convertConfig := &Config{}
	err := convertConfig.Load(&ctx, l, cfg, "")
	if err == nil {
		t.Fatalf("Load() expected error for nonexistent database")
	}

	expectedError := "база данных NONEXISTENT_DB не найдена в конфигурации"
	if err.Error() != expectedError {
		t.Errorf("Load() error = %v, want %v", err.Error(), expectedError)
	}
}

// TestConfig_InitDb tests the InitDb method
func TestConfig_InitDb(t *testing.T) {
	ctx := context.Background()
	var programLevel = new(slog.LevelVar)
	programLevel.Set(slog.LevelDebug)
	l := getSlog(programLevel)

	cfg := &config.Config{}

	// Test with DbExist = true (should return early)
	convertConfig := &Config{
		OneDB: designer.OneDb{
			DbExist: true,
		},
	}
	err := convertConfig.InitDb(&ctx, l, cfg)
	if err != nil {
		t.Errorf("InitDb() with DbExist=true should not error, got %v", err)
	}

	// Test with ServerDb = true (should return early)
	convertConfig = &Config{
		OneDB: designer.OneDb{
			DbExist:  false,
			ServerDb: true,
		},
	}
	err = convertConfig.InitDb(&ctx, l, cfg)
	if err != nil {
		t.Errorf("InitDb() with ServerDb=true should not error, got %v", err)
	}
}

