start:
	go run ./cmd/app

test:
	go test -v -count=1 -race -shuffle=on ./...

download-tailwindcss:
	curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
	chmod +x tailwindcss-macos-arm64
	mv tailwindcss-macos-arm64 tailwindcss

css:
	./tailwindcss -i view/main.css -o public/main.css
