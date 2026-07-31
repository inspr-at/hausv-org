FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS build
WORKDIR /src
ARG APP_VERSION
ARG GIT_COMMIT=dev
COPY go.mod ./
RUN go mod download
COPY . .
RUN test -n "${APP_VERSION}" && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/markus-barta/hausv-org/internal/version.Version=${APP_VERSION} -X github.com/markus-barta/hausv-org/internal/version.Commit=${GIT_COMMIT}" -o /out/hausv-org ./cmd/hausv-org

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35
COPY --from=build /out/hausv-org /hausv-org
EXPOSE 8080
ENTRYPOINT ["/hausv-org"]
