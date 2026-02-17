package convert

import (
	"os"

	"github.com/Kargones/apk-ci/internal/config"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func mergeSetting(cfg *config.Config) (string, error) {
	tFile, err := os.CreateTemp(cfg.WorkDir, "*.xml")
	if err != nil {
		return "", err
	}
	if _, err := tFile.Write([]byte(MergeSettingsString)); err != nil {
		return "", err
	}
	if err := tFile.Close(); err != nil {
		return "", err
	}
	return tFile.Name(), nil
}

/*
	func (cc *Config) SourceData(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
		if len(cc.RepURL) > 10 && cc.RepURL[:4] == "http" {
			g := git.Git{}
			g.RepURL = cc.RepURL
			g.RepPath = path.Join(cfg.TmpDir, "r")
			g.Branch = cc.Branch
			g.CommitSHA1 = cc.CommitSHA1
			err := g.Clone(ctx, l)
			if err != nil {
				l.Error("Ошибка клонирования репозитория",
					slog.String("URL репозитория", g.RepURL),
					slog.String("Каталог приемник", g.RepPath),
					slog.String("Ветка", g.Branch),
					slog.String("Коммит", g.CommitSHA1),
					slog.String("Текст ошибки", err.Error()),
				)
				return err
			}
			cc.SourceRoot = g.RepPath
		} else if ok, _ := exists(cc.RepURL); ok {
			cc.SourceRoot = cc.RepURL
		} else {
			l.Error("Отсутствует источник данных",
				slog.String("URL репозитория", cc.RepURL),
			)
			return fmt.Errorf("отсутствует источник данных %v", cc.RepURL)
		}
		return nil
	}
*/
/*
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
*/
