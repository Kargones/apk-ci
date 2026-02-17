package store

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// ReadReport читает отчет из хранилища конфигурации.
// Функция формирует отчет о версиях конфигурации в хранилище,
// начиная с указанной версии, и возвращает записи, пользователей и максимальную версию.
// Параметры:
//   - l: логгер для записи сообщений
//   - dbConnectString: строка подключения к базе данных
//   - cfg: конфигурация приложения
//   - startVersion: начальная версия для формирования отчета
//
// Возвращает:
//   - []Record: список записей хранилища
//   - []User: список пользователей
//   - int: максимальная версия в хранилище
//   - error: ошибку, если операция не удалась
func (s *Store) ReadReport(ctx context.Context, l *slog.Logger, dbConnectString string, cfg *config.Config, startVersion int) ([]Record, []User, int, error) {
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryReport")
	repPath := filepath.Join(r.TmpDir, s.Name+".txt")
	r.Params = append(r.Params, repPath)
	r.Params = append(r.Params, "-NBegin")
	r.Params = append(r.Params, fmt.Sprint(startVersion))
	r.Params = append(r.Params, "-ReportFormat")
	r.Params = append(r.Params, "txt")
	if s.Name != "Main" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, s.Name)
	}
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Отчет хранилища "+s.Name)
	bOut, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка выполнения команды формирования отчета", slog.String("storeName", s.Name), slog.String("error", err.Error()))
		return nil, nil, 0, fmt.Errorf("ошибка формирования отчета хранилища %s: %w", s.Name, err)
	}
	if len(bOut) > 44 && string(bOut[3:45]) != "Отчет успешно построен" {
		l.Error("Ошибка формирования отчета", slog.String("storeName", s.Name), slog.String("output", string(bOut)))
	}
	records, userList, maxVersion, err := ParseReport(repPath)
	return records, userList, maxVersion, nil
}

// ParseReport парсит отчет хранилища конфигурации из файла.
// Функция читает файл отчета и извлекает информацию о версиях хранилища и пользователях.
// TODO: переделать на чтение из буфера байт.
// Параметры:
//   - path: путь к файлу отчета
//
// Возвращает:
//   - []Record: список записей версий хранилища
//   - []User: список пользователей
//   - int: максимальный номер версии
//   - error: ошибку, если операция не удалась
func ParseReport(path string) ([]Record, []User, int, error) {
	maxVersion := 0
	records := []Record{}
	userList := []User{}
	// Валидация пути к файлу
	if strings.Contains(path, "..") || path == "" {
		return nil, nil, 0, fmt.Errorf("небезопасный путь к файлу: %s", path)
	}
	// #nosec G304 - path is validated above
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}()
	sr := Record{}
	dc := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		if t == "Комментарий:" {
			scanner.Scan()
			comment := scanner.Text()
			sr.Comment = comment
			continue
		}
		as := strings.Split(t, "\t")
		if len(as) < 2 {
			continue
		}
		switch as[0] {
		case "Версия:":
			if sr.Version != 0 {
				records = append(records, sr)
				sr = Record{}
			}
			i, parseErr := strconv.Atoi(as[1])
			if parseErr != nil {
				continue
			}
			sr.Version = i
			maxVersion = i
		case "Версия конфигурации:":
			sr.ConfVersion = as[1]
		case "Пользователь:":
			sr.User = as[1]
			exist := false
			for _, v := range userList {
				if v.StoreUserName == as[1] {
					exist = true
					break
				}
			}
			if !exist {
				cu := User{}
				cu.StoreUserName = as[1]
				userList = append(userList, cu)
			}
		case "Дата создания:":
			dc = as[1]
		case "Время создания:":
			dc += " " + as[1]
			sr.Date, err = time.Parse("02.01.2006 15:04:05", dc)
			if err != nil {
				dc = ""
				continue
			}
		}
	}

	// Append the last record if it exists
	if sr.Version != 0 {
		records = append(records, sr)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, 0, fmt.Errorf("scanner error: %w", scanErr)
	}
	return records, userList, maxVersion, nil
}
