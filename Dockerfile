from golang:latest
WORKDIR /go/src/app

COPY ./*.go ./
COPY ./models/ ./models/

RUN go get github.com/gorilla/mux
RUN go get github.com/lib/pq
RUN go get golang.org/x/crypto/bcrypt
RUN go get github.com/gorilla/sessions

CMD go run *.go
