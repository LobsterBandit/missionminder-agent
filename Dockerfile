FROM golang:alpine as builder

WORKDIR /tmp/build

RUN apk add --no-cache git

COPY go.* .

RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' -a \
      -o missionminder-agent .

FROM gcr.io/distroless/static as final

COPY --from=builder /tmp/build/missionminder-agent /missionminder-agent

ENTRYPOINT ["/missionminder-agent"]
