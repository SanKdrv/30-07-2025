start:
	cd backend && \
		go mod tidy && \
		go build -o app.exe ./app/main.go && \
		cd .. && \
		./backend/app.exe
# 	go run backend/app/main.go