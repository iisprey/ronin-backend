FROM golang AS builder

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build

FROM scratch

WORKDIR /app
COPY --from=builder /go/src/app/ /app/
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["go main"]