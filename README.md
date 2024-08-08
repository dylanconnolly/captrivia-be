# go

A starter backend for captrivia in Go. Make sure to grab the questions from
the directory above.

The dockerfile is meant for development in case you prefer to develop inside
Docker (though especially on non-Linux machines developing outside Docker will
likely have significantly faster compilation times).  Just build & run with the following:

```bash
docker build . --tag captrivia-be
docker compose-up -d
```

or if you want to run the backend locally
```bash
export QUESTIONS_FILE_PATH="/full_path/to/file/questions.json"
docker compose up fe redis
go run main.go
```