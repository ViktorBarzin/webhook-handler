# Get infra cli
FROM viktorbarzin/infra:latest AS infra-cli

FROM golang:alpine as webhook-handler
RUN mkdir /app 
ADD . /app/
WORKDIR /app 
RUN go build -o main .
RUN adduser -S -D -H -h /app appuser
USER appuser

COPY --from=infra-cli /app/infra_cli /usr/bin/infra_cli
CMD ["./main"]
