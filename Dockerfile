FROM golang:1.16.0-alpine AS builder

WORKDIR /opt/buid

COPY ./*.go ./*.html ./go.mod ./go.sum ./
COPY static ./static

RUN apk add gcc musl-dev linux-headers
RUN go get
RUN go build

FROM alpine:3.14

ENV PORT=17422
ENV DOMAIN=satdress.com
ENV SECRET=askdbasjdhvakjvsdjasd
ENV SITE_OWNER_URL=https://t.me/fiatjaf
ENV SITE_OWNER_NAME=@fiatjaf
ENV SITE_NAME=Satdress

COPY --from=builder /opt/buid/satdress /usr/local/bin/

EXPOSE 17422

CMD ["satdress"]
