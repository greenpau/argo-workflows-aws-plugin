ARG BUILDER=golang:1.20
ARG BASE=ubuntu:kinetic

FROM ${BUILDER} as builder
ARG GOPROXY=direct

WORKDIR /workspace
COPY . .

RUN go mod download
RUN GOPROXY=${GOPROXY} CGO_ENABLED=0 go build -ldflags "-w -s" -o argo-workflows-aws-plugin

FROM ${BASE}

COPY --from=builder /workspace/argo-workflows-aws-plugin /usr/bin/argo-workflows-aws-plugin
RUN apt update -y && apt install ca-certificates -y
CMD ["argo-workflows-aws-plugin"]