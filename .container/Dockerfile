FROM golang:1.23-bookworm

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o eks-node-viewer ./cmd/eks-node-viewer

CMD ["./eks-node-viewer"]
