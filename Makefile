export GOTOOLCHAIN ?= local

.PHONY: test web build tidy demo

test:
	go test ./...

web:
	cd web && npm install && npm run build
	find internal/webembed/dist -mindepth 1 ! -name .gitkeep -delete
	cp -R web/dist/. internal/webembed/dist/

build: web
	go build -o craftsail-growth ./cmd/craftsail-growth

tidy:
	go mod tidy

# demo builds the dashboard with a fictional brand and 30 days of data in
# ./demo, then opens it. Sign in as demo / quillpad-demo-2026. DEMO_LANG=zh gives
# Chinese prompts and Chinese engines.
DEMO_LANG ?= en
demo: build
	rm -rf demo
	go run ./scripts/demo -lang $(DEMO_LANG) -db demo/data/craftsail-growth.db
	mkdir -p demo/config && cp config/default.example.toml demo/config/
	cd demo && ../craftsail-growth ui
