start:
	go run ./cmd/app

test:
	go test -v -count=1 -race -shuffle=on ./...

css:
	./tailwindcss -i view/main.css -o public/main.css
