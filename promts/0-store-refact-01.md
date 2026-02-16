Добавь в конец функций (перед последним return):
func (s *Store) Bind(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, isMain bool) error {
func (s *Store) BindAdd(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {

обновление конфигурации из хранилища с помощью команды /ConfigurationRepositoryUpdateCfg используя параметры -revised и  -force.
обновление должно применяться к основной конфигурации и расширениям, если они указаны.
В качестве референса использую основной код функции Bind и документацию к команде /ConfigurationRepositoryUpdateCfg.