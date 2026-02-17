package rac

import "strings"

// sanitizeArgs маскирует пароли в аргументах для логирования.
// Имена пользователей НЕ маскируются — они не являются конфиденциальными данными
// и полезны для отладки проблем с аутентификацией.
func sanitizeArgs(args []string) []string {
	sanitized := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "--cluster-pwd=") ||
			strings.HasPrefix(arg, "--infobase-pwd=") {
			eqIdx := strings.Index(arg, "=")
			sanitized[i] = arg[:eqIdx+1] + "***"
		} else {
			sanitized[i] = arg
		}
	}
	return sanitized
}

// sanitizeString маскирует пароли в произвольной строке (например, stderr от RAC).
// Используется для предотвращения утечки credentials в сообщениях об ошибках.
func sanitizeString(s string) string {
	// Маскируем значения после --cluster-pwd= и --infobase-pwd=
	// Паттерн: --xxx-pwd=VALUE где VALUE продолжается до пробела, кавычки или конца строки
	result := s
	for _, prefix := range []string{"--cluster-pwd=", "--infobase-pwd="} {
		offset := 0
		for {
			idx := strings.Index(result[offset:], prefix)
			if idx == -1 {
				break
			}
			// Корректируем индекс относительно всей строки
			idx += offset
			// Находим конец значения пароля
			start := idx + len(prefix)
			end := start
			for end < len(result) {
				ch := result[end]
				// Пароль заканчивается на пробеле, кавычке, или конце строки
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '"' || ch == '\'' {
					break
				}
				end++
			}
			// Заменяем пароль на ***
			result = result[:start] + "***" + result[end:]
			// Продвигаем offset за обработанную область
			offset = start + 3 // len("***") = 3
		}
	}
	return result
}

// clusterAuthArgs возвращает аргументы аутентификации кластера.
func (c *racClient) clusterAuthArgs() []string {
	var args []string
	if c.clusterUser != "" {
		args = append(args, "--cluster-user="+c.clusterUser)
	}
	if c.clusterPass != "" {
		args = append(args, "--cluster-pwd="+c.clusterPass)
	}
	return args
}

// infobaseAuthArgs возвращает аргументы аутентификации информационной базы.
func (c *racClient) infobaseAuthArgs() []string {
	var args []string
	if c.infobaseUser != "" {
		args = append(args, "--infobase-user="+c.infobaseUser)
	}
	if c.infobasePass != "" {
		args = append(args, "--infobase-pwd="+c.infobasePass)
	}
	return args
}
