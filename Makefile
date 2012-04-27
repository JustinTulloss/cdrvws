build:
	go get
	go install

serve: build
	foreman start

deploy:
	git push heroku master
