FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /overwatch ./cmd/overwatch

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /overwatch /overwatch
WORKDIR /
EXPOSE 3030
ENTRYPOINT ["/overwatch"]
CMD ["serve"]
