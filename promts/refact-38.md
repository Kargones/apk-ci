Создай функцию которая устанавливает состояние переданных в массиве структур файлов в репозиторий на сервере gitea с помощью gitea API. Используй в качестве референса следующий curl запрос:



ChangeFilesOptions{
description:	
ChangeFilesOptions options for creating, updating or deleting multiple files
Note: author and committer are optional (if only one is given, it will be used for the other, otherwise the authenticated user will be used)

author	Identity{
description:	
Identity for a person's identity like an author or committer

email	string($email)
name	string
}
branch	string
branch (optional) to base this file from. if not given, the default branch is used

committer	Identity{
description:	
Identity for a person's identity like an author or committer

email	string($email)
name	string
}
dates	CommitDateOptions{
description:	
CommitDateOptions store dates for GIT_AUTHOR_DATE and GIT_COMMITTER_DATE

author	string($date-time)
committer	string($date-time)
}
files*	[
x-go-name: Files
list of file operations

ChangeFileOperation{
description:	
ChangeFileOperation for creating, updating or deleting a file

content	string
new or updated file content, must be base64 encoded

from_path	string
old path of the file to move

operation*	string
indicates what to do with the file

Enum:
[ create, update, delete ]
path*	string
path to the existing or new file

sha	string
sha is the SHA for the file that already exists, required for update or delete

}]
message	string
message (optional) for the commit of this file. if not supplied, a default message will be used

new_branch	string
new_branch (optional) will make a new branch from branch before creating the file

signoff	boolean
Add a Signed-off-by trailer by the committer at the end of the commit log message.

}

```bash
  'https://gitea.example.com/api/v1/repos/test/toir-100/contents' \
  -H 'Authorization: token YOUR_TOKEN' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
    "branch": "main",
    "author": {
      "name": "DevOps Bot",
      "email": "devops@example.com"
    },
    "committer": {
      "name": "DevOps Bot",
      "email": "devops@example.com"
    },
    "commit_message": "Обновление конфигурации: добавлены и удалены файлы",
    "operations": [
      {
        "operation": "create",
        "path": "new_folder/new_file.txt",
        "content": "Новый файл\n"
      },
      {
        "operation": "update",
        "path": "config/settings.json",
        "content": "{ \"param\": \"new value\" }\n",
        "sha": "старый_sha_файла"
      },
      {
        "operation": "delete",
        "path": "old_folder/old_file.txt",
        "sha": "sha_удаляемого_файла"
      }
    ]
  }'
```