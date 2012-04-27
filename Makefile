build:
	go get
	go install

serve: build
	foreman start
