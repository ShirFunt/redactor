# Stage 1: build the binary with the Go toolchain.
# Use the same Go version as the "go" line in go.mod.
FROM golang:1.27 AS build
WORKDIR /src

# Copy the dependency files first, so this layer is cached
# until the dependencies change, not on every code change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 makes a static binary that runs without libc.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/redactor .

# Stage 2: a minimal runtime image. It has no shell and no package manager.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/redactor /usr/local/bin/redactor
# Default rules. In Kubernetes, a ConfigMap mounted at the same path replaces this.
COPY rules.yaml /etc/redactor/rules.yaml

EXPOSE 8080
# "nonroot" runs the process as a non-root user.
ENTRYPOINT ["/usr/local/bin/redactor"]
CMD ["--serve", "--addr", ":8080", "--rules", "/etc/redactor/rules.yaml"]