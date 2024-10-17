FROM golang:1.23-bookworm as build

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o eks-node-viewer ./cmd/eks-node-viewer

FROM debian:buster-slim

WORKDIR /app

COPY --from=build /app/eks-node-viewer /app/eks-node-viewer

CMD ["./eks-node-viewer"]