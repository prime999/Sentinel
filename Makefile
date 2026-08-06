.PHONY: web build run clean install

web:
	cd web && npm install && npm run build

build: web
	go build -o bin/sentinel ./cmd/sentinel

run: build
	./bin/sentinel -config config.example.yaml

clean:
	rm -rf bin internal/ui/dist/* web/node_modules web/dist
