FROM golang:1.24-alpine AS build
WORKDIR /src
ARG APP_VERSION=0.6.13
ARG GIT_COMMIT=dev
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.appVersion=${APP_VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /out/weg-portal .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/weg-portal /weg-portal
EXPOSE 8080
ENTRYPOINT ["/weg-portal"]
