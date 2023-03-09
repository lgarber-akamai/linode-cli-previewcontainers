FROM python:3.11.2-alpine3.17

RUN apk add git make

WORKDIR /scripts

COPY run.sh .

ENTRYPOINT ["./run.sh"]