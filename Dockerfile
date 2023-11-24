FROM golang:1.21-alpine3.18

RUN apk add vips-dev gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN mkdir /geonow
RUN CGO_ENABLED=1 GOOS=linux go build -o /geonow/app

WORKDIR /geonow
RUN rm -rf /build
EXPOSE 8080
CMD ["/geonow/app"]
