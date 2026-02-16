Добавь в функцию EnableServiceMode следующий функционал:
- переменную message вынеси в константы
- перед применением кода который идет за:
    // Включаем сервисный режим
-- получаем текущее состояние     --scheduled-jobs-deny
-- если оно равно "on" то значение --denied-message=message + "."
-- иначе значение --denied-message=message
Добавь в функцию DisableServiceMode следующий функционал:
-- в начале функции получаем текущее состояние     --scheduled-jobs-deny
-- если значение равно message + "." то в параметрах не передаем "--scheduled-jobs-deny=off"
-- иначе в параметрах передаем "--scheduled-jobs-deny=off"


