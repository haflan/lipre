ARG build_goarch=amd64
ARG build_goarm=8

### Builder stage
FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git npm
COPY . /src/

# Build static website files
WORKDIR /src/ui
RUN mkdir -p /static
RUN npm install; \
    npm run build;

# Build binary
WORKDIR /src/
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=$build_goarch
ENV GOARM=$build_goarm
RUN go get -d ./... ; \
    go build -ldflags="-w -s" -o /go/bin/lipre .

### Production image
FROM scratch
COPY --from=builder /go/bin/lipre /lipre
COPY --from=builder /src/ui/dist /ui/dist
ENTRYPOINT ["/lipre"]
# Replace the above with this when lipre is modified to take static dir arg
#COPY --from=builder /src/ui/dist /static
#ENTRYPOINT ["/lipre", "-s", "/static"] # uncomment when implemented
