LINTER := golangci-lint
.PHONY: lint fmt ci test devdeps
up:
	docker compose up -d

down:
	docker compose down
	docker compose rm -f

clean:
	rm -rf docker/dynamodb

reset: down clean up

ci: devdeps lint test
lint:
	@echo ">> Running linter ($(LINTER))"
	$(LINTER) run

fmt:
	@echo ">> Formatting code"
	gofmt -w .
	goimports -w .

test:
	@echo ">> Running tests"
	go test -v -cover ./...

devdeps:
	@echo ">> Installing development dependencies"
	which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
