generate:
	sql-migrate up -env="production" -config configs/sql-migrate/metadata.yaml
	go generate ./...

clean:
	rm -rf internal/db
