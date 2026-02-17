// Package mssql предоставляет реализацию клиента для работы с Microsoft SQL Server.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	// blank import для драйвера SQL Server
	_ "github.com/denisenkom/go-mssqldb"
)

// dbaDatabase — имя базы данных DBA для получения статистики восстановления.
// Таблица BackupRequestJournal находится в схеме [DBA].[dbo].
// ВАЖНО: Если структура отличается на разных серверах, потребуется конфигурация.
//
// SECURITY NOTE (C-3): Эта константа используется в fmt.Sprintf для формирования SQL запроса.
// Это безопасно только потому, что значение является compile-time константой.
// НЕ ИЗМЕНЯТЬ на переменную без перехода на параметризованный запрос или квотирование!
const dbaDatabase = "DBA"

// Compile-time проверка реализации интерфейса
var _ Client = (*client)(nil)

// ClientOptions содержит параметры для создания MSSQL клиента.
type ClientOptions struct {
	// Server — адрес сервера MSSQL
	Server string
	// Port — порт сервера (по умолчанию 1433)
	Port int
	// User — имя пользователя
	User string
	// Password — пароль пользователя
	Password string
	// Database — имя базы данных для подключения (обычно "master")
	Database string
	// Timeout — таймаут подключения
	Timeout time.Duration
	// Encrypt — использовать TLS шифрование (по умолчанию true для безопасности)
	// ВАЖНО: Если Encrypt=false и encryptSet=false, будет использовано значение true по умолчанию.
	// Для явного отключения шифрования используйте NewClientWithEncrypt(opts, false).
	Encrypt bool
	// encryptSet — внутренний флаг, указывающий что Encrypt был явно задан (M4 fix)
	// Это приватное поле, не экспортируется. Для явного контроля шифрования
	// используйте конструктор NewClientWithEncrypt вместо NewClient.
	encryptSet bool
}

// client — реализация интерфейса Client для MSSQL.
type client struct {
	db   *sql.DB
	opts ClientOptions
}

// NewClient создаёт новый MSSQL клиент с указанными параметрами.
// Примечание: подключение устанавливается отложенно при первом запросе или через Connect().
func NewClient(opts ClientOptions) (Client, error) {
	// Валидация обязательных параметров (M5 fix)
	if opts.Server == "" {
		return nil, fmt.Errorf("%s: server is required", ErrMSSQLConnect)
	}
	// Установка значений по умолчанию
	if opts.Port == 0 {
		opts.Port = 1433
	}
	// Валидация порта (M6 fix)
	if opts.Port < 1 || opts.Port > 65535 {
		return nil, fmt.Errorf("%s: invalid port %d, must be between 1 and 65535", ErrMSSQLConnect, opts.Port)
	}
	if opts.Database == "" {
		opts.Database = "master"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	// По умолчанию включаем шифрование для безопасности (H3 fix)
	// Если encryptSet=false, значит Encrypt не был явно задан — используем true
	if !opts.encryptSet {
		opts.Encrypt = true
	}

	return &client{
		opts: opts,
	}, nil
}

// NewClientWithEncrypt создаёт MSSQL клиент с явным указанием режима шифрования.
// Используйте этот конструктор для явного контроля над TLS.
func NewClientWithEncrypt(opts ClientOptions, encrypt bool) (Client, error) {
	opts.Encrypt = encrypt
	opts.encryptSet = true
	return NewClient(opts)
}

// Connect устанавливает соединение с сервером MSSQL.
func (c *client) Connect(ctx context.Context) error {
	// Определяем режим шифрования (H3 fix)
	encryptMode := "true"
	if !c.opts.Encrypt {
		encryptMode = "disable"
	}

	// Экранируем параметры для защиты от инъекций в connection string (H1 fix)
	// Используем url.QueryEscape для безопасного экранирования специальных символов
	connString := fmt.Sprintf(
		"server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=%s;connection timeout=%d",
		escapeConnStringParam(c.opts.Server),
		escapeConnStringParam(c.opts.User),
		escapeConnStringParam(c.opts.Password),
		c.opts.Port,
		escapeConnStringParam(c.opts.Database),
		encryptMode,
		int(c.opts.Timeout.Seconds()),
	)

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrMSSQLConnect, err)
	}

	// Проверяем подключение
	// H-1 fix: проверяем ctx.Err() для более точной диагностики причины ошибки
	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// best-effort close; original error is more important
		}
		if ctx.Err() != nil {
			return fmt.Errorf("%s: context cancelled during ping: %w", ErrMSSQLConnect, ctx.Err())
		}
		return fmt.Errorf("%s: ping failed: %w", ErrMSSQLConnect, err)
	}

	c.db = db
	return nil
}

// escapeConnStringParam экранирует параметр для безопасного использования в connection string.
// Защищает от инъекции управляющих символов (; = и др.) в DSN.
func escapeConnStringParam(s string) string {
	// go-mssqldb использует URL-подобный формат, где ; и = имеют особое значение
	// Экранируем эти символы через URL encoding
	return url.QueryEscape(s)
}

// Close закрывает соединение с сервером.
func (c *client) Close() error {
	if c.db != nil {
		err := c.db.Close()
		c.db = nil
		return err
	}
	return nil
}

// Ping проверяет доступность сервера.
func (c *client) Ping(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("%s: connection not established", ErrMSSQLConnect)
	}
	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("%s: %w", ErrMSSQLConnect, err)
	}
	return nil
}

// Restore выполняет восстановление базы данных из резервной копии
// через хранимую процедуру sp_DBRestorePSFromHistoryD.
func (c *client) Restore(ctx context.Context, opts RestoreOptions) error {
	if c.db == nil {
		return fmt.Errorf("%s: connection not established", ErrMSSQLRestore)
	}

	// Создаём контекст с таймаутом если указан
	execCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	query := `
	USE master;
	EXEC sp_DBRestorePSFromHistoryD
		@Description = @p1,
		@DayToRestore = @p2,
		@DomainUser = @p3,
		@SrcServer = @p4,
		@SrcDB = @p5,
		@DstServer = @p6,
		@DstDB = @p7;
	`

	_, err := c.db.ExecContext(execCtx, query,
		opts.Description,
		opts.TimeToRestore,
		opts.User,
		opts.SrcServer,
		opts.SrcDB,
		opts.DstServer,
		opts.DstDB,
	)
	if err != nil {
		// Проверяем, была ли ошибка из-за таймаута
		if execCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%s: operation timed out after %v", ErrMSSQLTimeout, opts.Timeout)
		}
		return fmt.Errorf("%s: %w", ErrMSSQLRestore, err)
	}

	return nil
}

// GetRestoreStats возвращает статистику операций восстановления
// из таблицы BackupRequestJournal.
func (c *client) GetRestoreStats(ctx context.Context, opts StatsOptions) (*RestoreStats, error) {
	if c.db == nil {
		return nil, fmt.Errorf("%s: connection not established", ErrMSSQLQuery)
	}

	// Используем константу dbaDatabase для имени базы (H3 fix)
	query := fmt.Sprintf(`
	SELECT
		AVG(DATEDIFF(SECOND, RequestDate, CompliteTime)),
		MAX(DATEDIFF(SECOND, RequestDate, CompliteTime))
	FROM [%s].[dbo].[BackupRequestJournal]
	WHERE CompliteTime IS NOT NULL
		AND RequestDate IS NOT NULL
		AND RequestDate >= @p1
		AND SrcDB = @p2
		AND DstServer = @p3;
	`, dbaDatabase)

	var avgTime, maxTime sql.NullInt64
	err := c.db.QueryRowContext(ctx, query,
		opts.TimeToStatistic,
		opts.SrcDB,
		opts.DstServer,
	).Scan(&avgTime, &maxTime)

	if err != nil {
		if err == sql.ErrNoRows {
			// Нет данных — возвращаем пустую статистику
			return &RestoreStats{
				HasData: false,
			}, nil
		}
		return nil, fmt.Errorf("%s: %w", ErrMSSQLQuery, err)
	}

	// HasData = true только когда оба значения (avg и max) доступны (M6 fix)
	// Это строже чем в GetBackupSize, т.к. для расчёта таймаута нужны оба значения
	stats := &RestoreStats{
		HasData: avgTime.Valid && maxTime.Valid,
	}
	if avgTime.Valid {
		stats.AvgRestoreTimeSec = avgTime.Int64
	}
	if maxTime.Valid {
		stats.MaxRestoreTimeSec = maxTime.Int64
	}

	return stats, nil
}

// GetBackupSize возвращает размер последней резервной копии базы данных в байтах.
// Примечание: в текущей реализации возвращает 0, так как эта информация
// не используется в основном workflow восстановления.
func (c *client) GetBackupSize(ctx context.Context, database string) (int64, error) {
	if c.db == nil {
		return 0, fmt.Errorf("%s: connection not established", ErrMSSQLQuery)
	}

	// Запрос размера последнего бэкапа из msdb
	query := `
	SELECT TOP 1
		bs.backup_size
	FROM msdb.dbo.backupset bs
	WHERE bs.database_name = @p1
		AND bs.type = 'D'
	ORDER BY bs.backup_finish_date DESC;
	`

	var size sql.NullInt64
	err := c.db.QueryRowContext(ctx, query, database).Scan(&size)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("%s: %w", ErrMSSQLQuery, err)
	}

	if size.Valid {
		return size.Int64, nil
	}
	return 0, nil
}

