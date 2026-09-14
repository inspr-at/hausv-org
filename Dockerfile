FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df AS build
WORKDIR /src
ARG TARGETARCH
ARG APP_VERSION
ARG GIT_COMMIT=dev
ARG RELEASE_CHANNEL=production
COPY go.mod ./
RUN go mod download
COPY . .
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate && \
    test -n "${APP_VERSION}" && \
    APP_VERSION="${APP_VERSION}" go run ./cmd/verify-release && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w -X github.com/inspr-at/hausv-org/internal/version.Version=${APP_VERSION} -X github.com/inspr-at/hausv-org/internal/version.Commit=${GIT_COMMIT} -X github.com/inspr-at/hausv-org/internal/version.Channel=${RELEASE_CHANNEL}" -o /out/hausv-org ./cmd/hausv-org && \
    mkdir -p /out/connectors && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X github.com/inspr-at/hausv-org/internal/version.Version=${APP_VERSION} -X github.com/inspr-at/hausv-org/internal/version.Commit=${GIT_COMMIT} -X github.com/inspr-at/hausv-org/internal/version.Channel=${RELEASE_CHANNEL}" -o /out/connectors/hausv-connector-linux-amd64 ./cmd/hausv-org && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w -X github.com/inspr-at/hausv-org/internal/version.Version=${APP_VERSION} -X github.com/inspr-at/hausv-org/internal/version.Commit=${GIT_COMMIT} -X github.com/inspr-at/hausv-org/internal/version.Channel=${RELEASE_CHANNEL}" -o /out/connectors/hausv-connector-linux-arm64 ./cmd/hausv-org

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35
COPY --from=build /out/hausv-org /hausv-org
COPY --from=build /out/connectors /connector-downloads
EXPOSE 8080
ENTRYPOINT ["/hausv-org"]
