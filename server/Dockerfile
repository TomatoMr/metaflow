FROM alpine:latest as metaflow-server

MAINTAINER songzhen@yunshan.net

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add tzdata

COPY ./server.yaml /etc/
RUN mkdir /etc/mysql
COPY ./controller/db/mysql/migrate/rawsql /etc/mysql
COPY ./querier/db_descriptions /etc/db_descriptions/
ARG TARGETARCH 

RUN --mount=target=/tmp-mount \
    cp -a /tmp-mount/bin/metaflow-server.${TARGETARCH} /bin/metaflow-server

CMD /bin/metaflow-server
